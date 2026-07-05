package spotify

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const testAccessToken = "spotify-access-token-secret"

type testTrack struct {
	ID string `json:"id"`
}

func TestGetJSONAddsSpotifyAccessToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get(headerAuthorization); got != "Bearer "+testAccessToken {
			t.Fatalf("Authorization header = %q", got)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"items":[{"id":"track-1"}]}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	var got Page[testTrack]
	err := client.GetJSON(context.Background(), "/v1/me/playlists", RequestOptions{
		AccessToken: testAccessToken,
	}, &got)
	if err != nil {
		t.Fatalf("GetJSON returned error: %v", err)
	}
	if len(got.Items) != 1 || got.Items[0].ID != "track-1" {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestGetJSONRequiresSpotifyAccessToken(t *testing.T) {
	client := newTestClient(t, "http://example.test")

	err := client.GetJSON(context.Background(), "/v1/me/playlists", RequestOptions{}, nil)
	if !errors.Is(err, ErrMissingAccessToken) {
		t.Fatalf("err = %v", err)
	}
}

func TestAPIErrorParsesPayloadAndRedactsAccessToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"status":401,"message":"bad spotify-access-token-secret"}}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	err := client.GetJSON(context.Background(), "/v1/me/playlists", RequestOptions{
		AccessToken: testAccessToken,
	}, nil)
	if err == nil {
		t.Fatal("GetJSON returned nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err type = %T", err)
	}
	if apiErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d", apiErr.StatusCode)
	}
	message := err.Error()
	if strings.Contains(message, testAccessToken) {
		t.Fatalf("error leaked access token: %s", message)
	}
	if !strings.Contains(message, "[redacted]") {
		t.Fatalf("error did not redact details: %s", message)
	}
}

func TestGetAllPagesFollowsNextLinks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.String() {
		case "/v1/me/playlists":
			_, _ = w.Write([]byte(`{"items":[{"id":"one"}],"next":"` + serverURL(r) + `/v1/me/playlists?offset=1"}`))
		case "/v1/me/playlists?offset=1":
			_, _ = w.Write([]byte(`{"items":[{"id":"two"}],"next":null}`))
		default:
			t.Fatalf("unexpected request URL: %s", r.URL.String())
		}
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	got, err := GetAllPages[testTrack](context.Background(), client, "/v1/me/playlists", RequestOptions{
		AccessToken: testAccessToken,
	})
	if err != nil {
		t.Fatalf("GetAllPages returned error: %v", err)
	}
	if len(got) != 2 || got[0].ID != "one" || got[1].ID != "two" {
		t.Fatalf("unexpected pages: %+v", got)
	}
}

func TestGetJSONRetriesRateLimitAndServerErrors(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"status":429,"message":"rate limited"}}`))
			return
		}
		if attempts == 2 {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`{"error":{"status":502,"message":"temporary"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"items":[{"id":"ok"}]}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	var got Page[testTrack]
	if err := client.GetJSON(context.Background(), "/v1/search?type=track&q=piano", RequestOptions{AccessToken: testAccessToken}, &got); err != nil {
		t.Fatalf("GetJSON returned error: %v", err)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d", attempts)
	}
	if len(got.Items) != 1 || got.Items[0].ID != "ok" {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestGetJSONDoesNotRetryPermanentClientErrors(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"status":400,"message":"bad request"}}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	err := client.GetJSON(context.Background(), "/v1/search?type=track", RequestOptions{AccessToken: testAccessToken}, nil)
	if err == nil {
		t.Fatal("GetJSON returned nil")
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d", attempts)
	}
}

func newTestClient(t *testing.T, baseURL string) *Client {
	t.Helper()
	client, err := New(Config{
		BaseURL:        baseURL,
		MaxRetries:     2,
		RetryBaseDelay: time.Nanosecond,
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	return client
}

func serverURL(r *http.Request) string {
	return "http://" + r.Host
}
