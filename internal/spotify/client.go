package spotify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultBaseURL      = "https://api.spotify.com"
	headerAuthorization = "Authorization"
)

var ErrMissingAccessToken = errors.New("spotify access token is required")

// Config contains the Spotify Web API client settings and test seams.
type Config struct {
	BaseURL        string
	HTTPClient     *http.Client
	MaxRetries     int
	RetryBaseDelay time.Duration
}

// Client wraps Spotify Web API HTTP behavior so handlers do not build raw upstream requests.
type Client struct {
	baseURL        *url.URL
	httpClient     *http.Client
	maxRetries     int
	retryBaseDelay time.Duration
}

// RequestOptions carries the per-request Spotify user access token required by Web API calls.
type RequestOptions struct {
	AccessToken string
}

// Page models Spotify collection responses that use items and next for pagination.
type Page[T any] struct {
	Items []T    `json:"items"`
	Next  string `json:"next"`
}

// APIError preserves Spotify error details while keeping access tokens out of messages.
type APIError struct {
	StatusCode   int
	SpotifyError ErrorObject
}

// ErrorObject is the regular Spotify Web API error shape.
type ErrorObject struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

// Error returns a compact diagnostic suitable for logs and REST error mapping.
func (e *APIError) Error() string {
	if e.SpotifyError.Message == "" {
		return fmt.Sprintf("spotify api error: status %d", e.StatusCode)
	}
	return fmt.Sprintf("spotify api error: status %d: %s", e.StatusCode, e.SpotifyError.Message)
}

// New validates client settings and creates a reusable Spotify Web API client.
func New(cfg Config) (*Client, error) {
	rawBaseURL := strings.TrimSpace(cfg.BaseURL)
	if rawBaseURL == "" {
		rawBaseURL = defaultBaseURL
	}
	baseURL, err := url.Parse(rawBaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse spotify base url: %w", err)
	}
	if baseURL.Scheme == "" || baseURL.Host == "" {
		return nil, fmt.Errorf("parse spotify base url: absolute URL required")
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	maxRetries := cfg.MaxRetries
	if maxRetries == 0 {
		maxRetries = 2
	}

	retryBaseDelay := cfg.RetryBaseDelay
	if retryBaseDelay == 0 {
		retryBaseDelay = 100 * time.Millisecond
	}

	return &Client{
		baseURL:        baseURL,
		httpClient:     httpClient,
		maxRetries:     maxRetries,
		retryBaseDelay: retryBaseDelay,
	}, nil
}

// BaseURL returns the configured upstream URL for diagnostics and tests.
func (c *Client) BaseURL() string {
	return c.baseURL.String()
}

// GetJSON sends an authenticated GET request and decodes a successful JSON response into out.
func (c *Client) GetJSON(ctx context.Context, path string, opts RequestOptions, out any) error {
	return c.doJSON(ctx, http.MethodGet, path, opts, nil, out)
}

// PostJSON sends an authenticated JSON POST request and decodes a successful JSON response into out.
func (c *Client) PostJSON(ctx context.Context, path string, opts RequestOptions, body any, out any) error {
	return c.doJSON(ctx, http.MethodPost, path, opts, body, out)
}

// DeleteJSON sends an authenticated JSON DELETE request and decodes a successful JSON response into out.
func (c *Client) DeleteJSON(ctx context.Context, path string, opts RequestOptions, body any, out any) error {
	return c.doJSON(ctx, http.MethodDelete, path, opts, body, out)
}

// GetAllPages follows Spotify next links and concatenates items from each page.
func GetAllPages[T any](ctx context.Context, c *Client, path string, opts RequestOptions) ([]T, error) {
	var all []T
	next := path
	for strings.TrimSpace(next) != "" {
		var page Page[T]
		if err := c.GetJSON(ctx, next, opts, &page); err != nil {
			return nil, err
		}
		all = append(all, page.Items...)
		next = page.Next
	}
	return all, nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, opts RequestOptions, body any, out any) error {
	token := strings.TrimSpace(opts.AccessToken)
	if token == "" {
		return ErrMissingAccessToken
	}

	requestBody, err := encodeJSONBody(body)
	if err != nil {
		return err
	}

	var lastErr error
	attempts := c.maxRetries + 1
	for attempt := 0; attempt < attempts; attempt++ {
		req, err := c.newRequest(ctx, method, path, token, requestBody)
		if err != nil {
			return err
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			if attempt < attempts-1 {
				if waitErr := c.wait(ctx, retryDelay(nil, attempt, c.retryBaseDelay)); waitErr != nil {
					return waitErr
				}
				continue
			}
			return err
		}

		body, readErr := io.ReadAll(resp.Body)
		closeErr := resp.Body.Close()
		if readErr != nil {
			return readErr
		}
		if closeErr != nil {
			return closeErr
		}

		if shouldRetry(resp.StatusCode) && attempt < attempts-1 {
			lastErr = apiError(resp.StatusCode, body, token)
			if waitErr := c.wait(ctx, retryDelay(resp, attempt, c.retryBaseDelay)); waitErr != nil {
				return waitErr
			}
			continue
		}

		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			return apiError(resp.StatusCode, body, token)
		}
		if out == nil || len(bytes.TrimSpace(body)) == 0 {
			return nil
		}
		if err := json.Unmarshal(body, out); err != nil {
			return fmt.Errorf("decode spotify response: %w", err)
		}
		return nil
	}

	return lastErr
}

func (c *Client) newRequest(ctx context.Context, method, path, accessToken string, body []byte) (*http.Request, error) {
	u, err := c.resolveURL(path)
	if err != nil {
		return nil, err
	}

	var requestBody io.Reader
	if body != nil {
		requestBody = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), requestBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set(headerAuthorization, "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func encodeJSONBody(body any) ([]byte, error) {
	if body == nil {
		return nil, nil
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("encode spotify request body: %w", err)
	}
	return encoded, nil
}

func (c *Client) resolveURL(path string) (*url.URL, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("spotify request path is required")
	}
	ref, err := url.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("parse spotify request path: %w", err)
	}
	return c.baseURL.ResolveReference(ref), nil
}

func (c *Client) wait(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}

	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func shouldRetry(status int) bool {
	return status == http.StatusTooManyRequests ||
		status == http.StatusInternalServerError ||
		status == http.StatusBadGateway ||
		status == http.StatusServiceUnavailable ||
		status == http.StatusGatewayTimeout
}

func retryDelay(resp *http.Response, attempt int, base time.Duration) time.Duration {
	if resp != nil {
		if wait := parseRetryAfter(resp.Header.Get("Retry-After"), time.Now()); wait > 0 {
			return wait
		}
	}
	return base * time.Duration(1<<attempt)
}

func parseRetryAfter(value string, now time.Time) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(value); err == nil {
		return time.Duration(seconds) * time.Second
	}
	if at, err := http.ParseTime(value); err == nil && at.After(now) {
		return at.Sub(now)
	}
	return 0
}

func apiError(status int, body []byte, accessToken string) error {
	var payload struct {
		Error ErrorObject `json:"error"`
	}
	if err := json.Unmarshal(body, &payload); err != nil || payload.Error.Message == "" {
		return &APIError{StatusCode: status}
	}

	payload.Error.Message = redactSecret(payload.Error.Message, accessToken)
	return &APIError{StatusCode: status, SpotifyError: payload.Error}
}

func redactSecret(value, secret string) string {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return value
	}
	return strings.ReplaceAll(value, secret, "[redacted]")
}
