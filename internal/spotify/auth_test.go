package spotify

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientCredentialsTokenSendsBasicAuthAndForm(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/token" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s", r.Method)
		}
		wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("client-id:client-secret"))
		if got := r.Header.Get("Authorization"); got != wantAuth {
			t.Fatalf("Authorization = %q", got)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if got := r.Form.Get("grant_type"); got != "client_credentials" {
			t.Fatalf("grant_type = %q", got)
		}
		_, _ = w.Write([]byte(`{"access_token":"app-token","token_type":"bearer","expires_in":3600}`))
	}))
	defer server.Close()

	client, err := NewAuthClient(AuthConfig{
		AccountsBaseURL: server.URL,
		ClientID:        "client-id",
		ClientSecret:    "client-secret",
	})
	if err != nil {
		t.Fatalf("NewAuthClient returned error: %v", err)
	}

	token, err := client.ClientCredentialsToken(context.Background())
	if err != nil {
		t.Fatalf("ClientCredentialsToken returned error: %v", err)
	}
	if token.AccessToken != "app-token" || token.TokenType != "bearer" || token.ExpiresIn != 3600 {
		t.Fatalf("unexpected token: %+v", token)
	}
}

func TestNewAuthClientRequiresClientCredentials(t *testing.T) {
	_, err := NewAuthClient(AuthConfig{AccountsBaseURL: "http://accounts.test"})
	if !errors.Is(err, ErrMissingClientCredentials) {
		t.Fatalf("err = %v", err)
	}
}

func TestClientCredentialsTokenRedactsClientSecretFromErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid_client","error_description":"bad client-secret"}`))
	}))
	defer server.Close()

	client, err := NewAuthClient(AuthConfig{
		AccountsBaseURL: server.URL,
		ClientID:        "client-id",
		ClientSecret:    "client-secret",
	})
	if err != nil {
		t.Fatalf("NewAuthClient returned error: %v", err)
	}

	_, err = client.ClientCredentialsToken(context.Background())
	if err == nil {
		t.Fatal("ClientCredentialsToken returned nil")
	}
	if strings.Contains(err.Error(), "client-secret") {
		t.Fatalf("error leaked client secret: %v", err)
	}
	if !strings.Contains(err.Error(), "[redacted]") {
		t.Fatalf("error did not redact secret: %v", err)
	}
}
