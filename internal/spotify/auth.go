package spotify

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const defaultAccountsBaseURL = "https://accounts.spotify.com"

var ErrMissingClientCredentials = errors.New("spotify client id and secret are required")

// AuthConfig contains Spotify Accounts service settings.
type AuthConfig struct {
	AccountsBaseURL string
	ClientID        string
	ClientSecret    string
	HTTPClient      *http.Client
}

// AuthClient wraps calls to Spotify Accounts token endpoints.
type AuthClient struct {
	accountsBaseURL *url.URL
	clientID        string
	clientSecret    string
	httpClient      *http.Client
}

// TokenResponse is the successful Spotify token response shape.
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope,omitempty"`
}

// AuthError preserves Spotify Accounts error details while redacting client secrets.
type AuthError struct {
	StatusCode  int
	Code        string
	Description string
}

func (e *AuthError) Error() string {
	if e.Description != "" {
		return fmt.Sprintf("spotify auth error: status %d: %s", e.StatusCode, e.Description)
	}
	if e.Code != "" {
		return fmt.Sprintf("spotify auth error: status %d: %s", e.StatusCode, e.Code)
	}
	return fmt.Sprintf("spotify auth error: status %d", e.StatusCode)
}

// NewAuthClient validates Spotify Accounts settings and creates a reusable token client.
func NewAuthClient(cfg AuthConfig) (*AuthClient, error) {
	rawBaseURL := strings.TrimSpace(cfg.AccountsBaseURL)
	if rawBaseURL == "" {
		rawBaseURL = defaultAccountsBaseURL
	}
	accountsBaseURL, err := url.Parse(rawBaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse spotify accounts base url: %w", err)
	}
	if accountsBaseURL.Scheme == "" || accountsBaseURL.Host == "" {
		return nil, fmt.Errorf("parse spotify accounts base url: absolute URL required")
	}
	if strings.TrimSpace(cfg.ClientID) == "" || strings.TrimSpace(cfg.ClientSecret) == "" {
		return nil, ErrMissingClientCredentials
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &AuthClient{
		accountsBaseURL: accountsBaseURL,
		clientID:        strings.TrimSpace(cfg.ClientID),
		clientSecret:    strings.TrimSpace(cfg.ClientSecret),
		httpClient:      httpClient,
	}, nil
}

// ClientCredentialsToken requests an app-only Spotify access token.
func (c *AuthClient) ClientCredentialsToken(ctx context.Context) (TokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "client_credentials")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.accountsBaseURL.ResolveReference(&url.URL{Path: "/api/token"}).String(), strings.NewReader(form.Encode()))
	if err != nil {
		return TokenResponse{}, err
	}
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(c.clientID+":"+c.clientSecret)))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return TokenResponse{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return TokenResponse{}, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return TokenResponse{}, authError(resp.StatusCode, body, c.clientSecret)
	}

	var token TokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		return TokenResponse{}, fmt.Errorf("decode spotify auth response: %w", err)
	}
	return token, nil
}

func authError(status int, body []byte, clientSecret string) error {
	var payload struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &payload); err != nil || (payload.Error == "" && payload.ErrorDescription == "") {
		return &AuthError{StatusCode: status}
	}
	return &AuthError{
		StatusCode:  status,
		Code:        redactSecret(payload.Error, clientSecret),
		Description: redactSecret(payload.ErrorDescription, clientSecret),
	}
}
