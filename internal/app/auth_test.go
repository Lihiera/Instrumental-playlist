package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestSaveUserTokenStoresMetadataWithoutReturningRefreshToken(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/tokens", strings.NewReader(`{
		"access_token":"user-access-token",
		"refresh_token":"user-refresh-token",
		"token_type":"bearer",
		"scope":"playlist-read-private",
		"expires_in":3600
	}`))
	req.Header.Set("Content-Type", "application/json")

	Handler(Config{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "user-access-token") || strings.Contains(rec.Body.String(), "user-refresh-token") {
		t.Fatalf("response leaked token: %s", rec.Body.String())
	}

	var got tokenMetadata
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.ID == "" || !got.HasRefreshToken || got.Scope != "playlist-read-private" {
		t.Fatalf("unexpected token metadata: %+v", got)
	}
}

func TestSavedTokenMetadataCanBeReadFromMemory(t *testing.T) {
	router := Handler(Config{})
	save := httptest.NewRecorder()
	saveReq := httptest.NewRequest(http.MethodPost, "/v1/auth/tokens", strings.NewReader(`{"access_token":"user-access-token"}`))
	saveReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(save, saveReq)
	if save.Code != http.StatusCreated {
		t.Fatalf("save status = %d body = %s", save.Code, save.Body.String())
	}

	var saved tokenMetadata
	if err := json.Unmarshal(save.Body.Bytes(), &saved); err != nil {
		t.Fatalf("decode save response: %v", err)
	}

	get := httptest.NewRecorder()
	getReq := httptest.NewRequest(http.MethodGet, "/v1/auth/tokens/"+saved.ID, nil)
	router.ServeHTTP(get, getReq)
	if get.Code != http.StatusOK {
		t.Fatalf("get status = %d body = %s", get.Code, get.Body.String())
	}
	if strings.Contains(get.Body.String(), "user-access-token") {
		t.Fatalf("metadata response leaked token: %s", get.Body.String())
	}
}

func TestAuthStatusReturnsLoggedOutWhenNoTokenIsStored(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/auth/status", nil)

	Handler(Config{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	var got authStatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.LoggedIn || got.Token != nil || got.AccessTokenExpired {
		t.Fatalf("unexpected status response: %+v", got)
	}
}

func TestAuthStatusReturnsLatestTokenMetadataWithoutLeakingSecrets(t *testing.T) {
	router := Handler(Config{})
	for _, body := range []string{
		`{"access_token":"older-access-token","refresh_token":"older-refresh-token","token_type":"bearer","scope":"playlist-read-private","expires_in":3600}`,
		`{"access_token":"latest-access-token","refresh_token":"latest-refresh-token","token_type":"Bearer","scope":"playlist-read-private playlist-modify-private","expires_in":3600}`,
	} {
		save := httptest.NewRecorder()
		saveReq := httptest.NewRequest(http.MethodPost, "/v1/auth/tokens", strings.NewReader(body))
		saveReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(save, saveReq)
		if save.Code != http.StatusCreated {
			t.Fatalf("save status = %d body = %s", save.Code, save.Body.String())
		}
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/auth/status", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, secret := range []string{"older-access-token", "older-refresh-token", "latest-access-token", "latest-refresh-token"} {
		if strings.Contains(body, secret) {
			t.Fatalf("status response leaked secret %q: %s", secret, body)
		}
	}
	var got authStatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !got.LoggedIn || got.Token == nil {
		t.Fatalf("unexpected logged-out response: %+v", got)
	}
	if got.Token.ID == "" || got.Token.TokenType != "Bearer" || got.Token.Scope != "playlist-read-private playlist-modify-private" || !got.Token.HasRefreshToken {
		t.Fatalf("unexpected token metadata: %+v", got.Token)
	}
	if got.AccessTokenExpired {
		t.Fatalf("AccessTokenExpired = true")
	}
}

func TestLogoutClearsStoredTokensWithoutLeakingSecrets(t *testing.T) {
	router := Handler(Config{})
	save := httptest.NewRecorder()
	saveReq := httptest.NewRequest(http.MethodPost, "/v1/auth/tokens", strings.NewReader(`{
		"access_token":"logout-access-token",
		"refresh_token":"logout-refresh-token",
		"token_type":"Bearer",
		"scope":"playlist-read-private",
		"expires_in":3600
	}`))
	saveReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(save, saveReq)
	if save.Code != http.StatusCreated {
		t.Fatalf("save status = %d body = %s", save.Code, save.Body.String())
	}

	logout := httptest.NewRecorder()
	logoutReq := httptest.NewRequest(http.MethodPost, "/v1/auth/logout", nil)
	router.ServeHTTP(logout, logoutReq)
	if logout.Code != http.StatusOK {
		t.Fatalf("logout status = %d body = %s", logout.Code, logout.Body.String())
	}
	for _, secret := range []string{"logout-access-token", "logout-refresh-token"} {
		if strings.Contains(logout.Body.String(), secret) {
			t.Fatalf("logout response leaked secret %q: %s", secret, logout.Body.String())
		}
	}
	var logoutBody logoutResponse
	if err := json.Unmarshal(logout.Body.Bytes(), &logoutBody); err != nil {
		t.Fatalf("decode logout response: %v", err)
	}
	if !logoutBody.LoggedOut {
		t.Fatalf("LoggedOut = false")
	}

	status := httptest.NewRecorder()
	statusReq := httptest.NewRequest(http.MethodGet, "/v1/auth/status", nil)
	router.ServeHTTP(status, statusReq)
	if status.Code != http.StatusOK {
		t.Fatalf("status code = %d body = %s", status.Code, status.Body.String())
	}
	var got authStatusResponse
	if err := json.Unmarshal(status.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode status response: %v", err)
	}
	if got.LoggedIn || got.Token != nil {
		t.Fatalf("unexpected status after logout: %+v", got)
	}
}

func TestLogoutWithoutStoredTokenIsIdempotent(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/logout", nil)

	Handler(Config{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	var got logoutResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.LoggedOut {
		t.Fatalf("LoggedOut = true")
	}
}

func TestSpotifyOAuthLoginRedirectsWithStateAndPlaylistScopes(t *testing.T) {
	router := Handler(Config{
		SpotifyAccountsBaseURL: "http://accounts.test",
		SpotifyClientID:        "client-id",
		SpotifyClientSecret:    "client-secret",
		SpotifyRedirectURI:     "http://127.0.0.1:8080/oauth/spotify/callback",
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/oauth/spotify/login", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	location := rec.Header().Get("Location")
	u, err := url.Parse(location)
	if err != nil {
		t.Fatalf("parse Location: %v", err)
	}
	if u.Scheme != "http" || u.Host != "accounts.test" || u.Path != "/authorize" {
		t.Fatalf("unexpected redirect location: %s", location)
	}
	q := u.Query()
	if q.Get("response_type") != "code" || q.Get("client_id") != "client-id" {
		t.Fatalf("unexpected auth query: %s", location)
	}
	if q.Get("redirect_uri") != "http://127.0.0.1:8080/oauth/spotify/callback" {
		t.Fatalf("redirect_uri = %q", q.Get("redirect_uri"))
	}
	if q.Get("state") == "" {
		t.Fatalf("state missing from redirect: %s", location)
	}
	for _, scope := range spotifyOAuthScopes {
		if !strings.Contains(q.Get("scope"), scope) {
			t.Fatalf("scope %q missing from redirect: %s", scope, location)
		}
	}
	if strings.Contains(location, "client-secret") {
		t.Fatalf("redirect leaked client secret: %s", location)
	}
}

func TestSpotifyOAuthCallbackValidatesStateExchangesCodeAndStoresMetadata(t *testing.T) {
	const (
		accessToken  = "oauth-user-access-token"
		refreshToken = "oauth-user-refresh-token"
		clientSecret = "client-secret"
	)
	var sawTokenExchange bool
	accounts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawTokenExchange = true
		if r.URL.Path != "/api/token" {
			t.Fatalf("accounts path = %s", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		for key, want := range map[string]string{
			"grant_type":   "authorization_code",
			"code":         "spotify-code",
			"redirect_uri": "http://127.0.0.1:8080/oauth/spotify/callback",
		} {
			if got := r.Form.Get(key); got != want {
				t.Fatalf("%s = %q", key, got)
			}
		}
		_, _ = w.Write([]byte(`{"access_token":"` + accessToken + `","refresh_token":"` + refreshToken + `","token_type":"bearer","expires_in":3600,"scope":"playlist-read-private playlist-modify-private"}`))
	}))
	defer accounts.Close()

	router := Handler(Config{
		SpotifyAccountsBaseURL: accounts.URL,
		SpotifyClientID:        "client-id",
		SpotifyClientSecret:    clientSecret,
		SpotifyRedirectURI:     "http://127.0.0.1:8080/oauth/spotify/callback",
	})

	login := httptest.NewRecorder()
	router.ServeHTTP(login, httptest.NewRequest(http.MethodGet, "/oauth/spotify/login", nil))
	state := stateFromLoginRedirect(t, login.Header().Get("Location"))

	callback := httptest.NewRecorder()
	callbackReq := httptest.NewRequest(http.MethodGet, "/oauth/spotify/callback?code=spotify-code&state="+url.QueryEscape(state), nil)
	router.ServeHTTP(callback, callbackReq)

	if callback.Code != http.StatusOK {
		t.Fatalf("callback status = %d body = %s", callback.Code, callback.Body.String())
	}
	if !sawTokenExchange {
		t.Fatal("token exchange was not requested")
	}
	body := callback.Body.String()
	for _, secret := range []string{accessToken, refreshToken, clientSecret} {
		if strings.Contains(body, secret) {
			t.Fatalf("callback response leaked secret %q: %s", secret, body)
		}
	}
	if got := callback.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/html") {
		t.Fatalf("Content-Type = %q", got)
	}
	for _, want := range []string{
		"Spotify login complete",
		"User token metadata was saved in process memory",
		"playlist-read-private playlist-modify-private",
		"Saved",
		"/v1/auth/tokens/",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("callback success page missing %q: %s", want, body)
		}
	}

	replay := httptest.NewRecorder()
	replayReq := httptest.NewRequest(http.MethodGet, "/oauth/spotify/callback?code=spotify-code&state="+url.QueryEscape(state), nil)
	router.ServeHTTP(replay, replayReq)
	if replay.Code != http.StatusBadRequest {
		t.Fatalf("replay status = %d body = %s", replay.Code, replay.Body.String())
	}
	assertErrorCode(t, replay.Body.String(), "spotify_oauth_state_invalid")
}

func TestSpotifyOAuthCallbackRejectsMissingCode(t *testing.T) {
	router := Handler(Config{
		SpotifyAccountsBaseURL: "http://accounts.test",
		SpotifyClientID:        "client-id",
		SpotifyClientSecret:    "client-secret",
		SpotifyRedirectURI:     "http://127.0.0.1:8080/oauth/spotify/callback",
	})

	login := httptest.NewRecorder()
	router.ServeHTTP(login, httptest.NewRequest(http.MethodGet, "/oauth/spotify/login", nil))
	state := stateFromLoginRedirect(t, login.Header().Get("Location"))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/oauth/spotify/callback?state="+url.QueryEscape(state), nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	assertErrorCode(t, rec.Body.String(), "spotify_oauth_code_missing")
}

func TestSpotifyOAuthCallbackRedactsSecretFromTokenExchangeErrors(t *testing.T) {
	const clientSecret = "client-secret"
	accounts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid_grant","error_description":"bad ` + clientSecret + `"}`))
	}))
	defer accounts.Close()

	router := Handler(Config{
		SpotifyAccountsBaseURL: accounts.URL,
		SpotifyClientID:        "client-id",
		SpotifyClientSecret:    clientSecret,
		SpotifyRedirectURI:     "http://127.0.0.1:8080/oauth/spotify/callback",
	})

	login := httptest.NewRecorder()
	router.ServeHTTP(login, httptest.NewRequest(http.MethodGet, "/oauth/spotify/login", nil))
	state := stateFromLoginRedirect(t, login.Header().Get("Location"))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/oauth/spotify/callback?code=bad-code&state="+url.QueryEscape(state), nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), clientSecret) {
		t.Fatalf("error response leaked client secret: %s", rec.Body.String())
	}
	assertErrorCode(t, rec.Body.String(), "spotify_auth_error")
}

func stateFromLoginRedirect(t *testing.T, location string) string {
	t.Helper()
	u, err := url.Parse(location)
	if err != nil {
		t.Fatalf("parse Location: %v", err)
	}
	state := u.Query().Get("state")
	if state == "" {
		t.Fatalf("state missing from Location: %s", location)
	}
	return state
}
