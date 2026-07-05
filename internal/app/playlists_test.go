package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const handlerTestToken = "handler-spotify-token-secret"

func TestPlaylistsEndpointRequiresBearerToken(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/playlists", nil)

	Handler(Config{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), handlerTestToken) {
		t.Fatalf("response leaked token-shaped value: %s", rec.Body.String())
	}
	assertErrorCode(t, rec.Body.String(), "missing_spotify_access_token")
}

func TestPlaylistsEndpointReturnsPaginatedSpotifyItems(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearer(t, r)
		switch r.URL.String() {
		case "/v1/me/playlists":
			_, _ = w.Write([]byte(`{"items":[{"id":"one"}],"next":"` + serverURL(r) + `/v1/me/playlists?offset=1"}`))
		case "/v1/me/playlists?offset=1":
			_, _ = w.Write([]byte(`{"items":[{"id":"two"}],"next":null}`))
		default:
			t.Fatalf("unexpected spotify path: %s", r.URL.String())
		}
	}))
	defer server.Close()

	rec := performSpotifyRequest(t, server.URL, http.MethodGet, "/v1/playlists", "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"id":"one"`) || !strings.Contains(rec.Body.String(), `"id":"two"`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestPlaylistsEndpointUsesStoredUserAccessTokenWhenAuthorizationHeaderIsMissing(t *testing.T) {
	const storedToken = "stored-user-access-token-secret"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer "+storedToken {
			t.Fatalf("Authorization = %q", got)
		}
		_, _ = w.Write([]byte(`{"items":[{"id":"from-stored-token"}],"next":null}`))
	}))
	defer server.Close()

	router := Handler(Config{SpotifyBaseURL: server.URL})
	save := httptest.NewRecorder()
	saveReq := httptest.NewRequest(http.MethodPost, "/v1/auth/tokens", strings.NewReader(`{"access_token":"`+storedToken+`","token_type":"Bearer","expires_in":3600}`))
	saveReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(save, saveReq)
	if save.Code != http.StatusCreated {
		t.Fatalf("save status = %d body = %s", save.Code, save.Body.String())
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/playlists", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"id":"from-stored-token"`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), storedToken) {
		t.Fatalf("response leaked stored token: %s", rec.Body.String())
	}
}

func TestPlaylistsEndpointPrefersAuthorizationHeaderOverStoredToken(t *testing.T) {
	const storedToken = "stored-user-access-token-secret"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearer(t, r)
		_, _ = w.Write([]byte(`{"items":[],"next":null}`))
	}))
	defer server.Close()

	router := Handler(Config{SpotifyBaseURL: server.URL})
	save := httptest.NewRecorder()
	saveReq := httptest.NewRequest(http.MethodPost, "/v1/auth/tokens", strings.NewReader(`{"access_token":"`+storedToken+`","expires_in":3600}`))
	saveReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(save, saveReq)
	if save.Code != http.StatusCreated {
		t.Fatalf("save status = %d body = %s", save.Code, save.Body.String())
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/playlists", nil)
	req.Header.Set("Authorization", "Bearer "+handlerTestToken)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func TestPlaylistsEndpointRejectsExpiredStoredUserAccessToken(t *testing.T) {
	tokens := newTokenStore()
	if _, err := tokens.Save(storedToken{
		AccessToken: "expired-user-access-token-secret",
		ExpiresAt:   time.Now().UTC().Add(-time.Minute),
	}); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	router := NewEngine()
	bindSpotifyHandlers(router, Config{SpotifyBaseURL: "http://spotify.test"}, tokens)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/playlists", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "expired-user-access-token-secret") {
		t.Fatalf("response leaked expired token: %s", rec.Body.String())
	}
	assertErrorCode(t, rec.Body.String(), "spotify_access_token_expired")
}

func TestCreatePlaylistUsesCurrentSpotifyUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearer(t, r)
		switch r.URL.Path {
		case "/v1/me":
			_, _ = w.Write([]byte(`{"id":"user 1"}`))
		case "/v1/users/user 1/playlists":
			if r.URL.EscapedPath() != "/v1/users/user%201/playlists" {
				t.Fatalf("escaped path = %s", r.URL.EscapedPath())
			}
			if r.Method != http.MethodPost {
				t.Fatalf("method = %s", r.Method)
			}
			var got createPlaylistRequest
			if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
				t.Fatalf("decode spotify request body: %v", err)
			}
			if got.Name != "Instrumental Mix" {
				t.Fatalf("name = %q", got.Name)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"created-playlist"}`))
		default:
			t.Fatalf("unexpected spotify path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	rec := performSpotifyRequest(t, server.URL, http.MethodPost, "/v1/playlists", `{"name":"Instrumental Mix"}`)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"id":"created-playlist"`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestPlaylistTracksEndpointUsesCurrentSpotifyPlaylistItemsAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearer(t, r)
		if r.URL.Path != "/v1/playlists/playlist-1/items" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"items":[{"track":{"id":"track-1"}}],"next":null}`))
	}))
	defer server.Close()

	rec := performSpotifyRequest(t, server.URL, http.MethodGet, "/v1/playlists/playlist-1/tracks", "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"id":"track-1"`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestAddPlaylistTracksRejectsMoreThanSpotifyBatchLimit(t *testing.T) {
	uris := make([]string, maxSpotifyPlaylistURIs+1)
	for i := range uris {
		uris[i] = "spotify:track:test"
	}
	body, err := json.Marshal(addTracksRequest{URIs: uris})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	rec := performSpotifyRequest(t, "http://spotify.test", http.MethodPost, "/v1/playlists/playlist-1/tracks", string(body))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	assertErrorCode(t, rec.Body.String(), "invalid_request")
}

func TestSearchTracksRequiresTerm(t *testing.T) {
	rec := performSpotifyRequest(t, "http://spotify.test", http.MethodGet, "/v1/search/tracks", "")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	assertErrorCode(t, rec.Body.String(), "invalid_request")
}

func TestSearchTracksMapsSpotifyErrorsWithoutLeakingAccessToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearer(t, r)
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"status":403,"message":"bad ` + handlerTestToken + `"}}`))
	}))
	defer server.Close()

	rec := performSpotifyRequest(t, server.URL, http.MethodGet, "/v1/search/tracks?term=piano", "")

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), handlerTestToken) {
		t.Fatalf("response leaked access token: %s", rec.Body.String())
	}
	assertErrorCode(t, rec.Body.String(), "spotify_api_error")
}

func TestNoLoginSearchPlaylistsUsesServerAppOnlyToken(t *testing.T) {
	const appOnlyToken = "app-only-token-secret"

	accounts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/token" {
			t.Fatalf("accounts path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"access_token":"` + appOnlyToken + `","token_type":"bearer","expires_in":3600}`))
	}))
	defer accounts.Close()

	spotifyAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer "+appOnlyToken {
			t.Fatalf("Authorization = %q", got)
		}
		if r.URL.String() != "/v1/search?type=playlist&limit=10&market=JP&q=focus" {
			t.Fatalf("URL = %s", r.URL.String())
		}
		_, _ = w.Write([]byte(`{"playlists":{"items":[{"id":"playlist-1"}],"next":null}}`))
	}))
	defer spotifyAPI.Close()

	router := Handler(Config{
		SpotifyAccountsBaseURL: accounts.URL,
		SpotifyBaseURL:         spotifyAPI.URL,
		SpotifyClientID:        "client-id",
		SpotifyClientSecret:    "client-secret",
	})

	searchRec := httptest.NewRecorder()
	searchReq := httptest.NewRequest(http.MethodGet, "/v1/noLogin/search/playlists?keyword=focus", nil)
	router.ServeHTTP(searchRec, searchReq)
	if searchRec.Code != http.StatusOK {
		t.Fatalf("search status = %d body = %s", searchRec.Code, searchRec.Body.String())
	}
	if !strings.Contains(searchRec.Body.String(), `"id":"playlist-1"`) {
		t.Fatalf("unexpected search body: %s", searchRec.Body.String())
	}
}

func TestNoLoginSearchPlaylistsRequiresKeyword(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/noLogin/search/playlists", nil)

	Handler(Config{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	assertErrorCode(t, rec.Body.String(), "invalid_request")
}

func TestDeletePlaylistTracksSendsExplicitTrackBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearer(t, r)
		if r.Method != http.MethodDelete {
			t.Fatalf("method = %s", r.Method)
		}
		if r.URL.Path != "/v1/playlists/playlist-1/tracks" {
			t.Fatalf("path = %s", r.URL.Path)
		}

		var got struct {
			Tracks []struct {
				URI string `json:"uri"`
			} `json:"tracks"`
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode spotify request body: %v", err)
		}
		if len(got.Tracks) != 1 || got.Tracks[0].URI != "spotify:track:one" {
			t.Fatalf("unexpected body: %+v", got)
		}
		_, _ = w.Write([]byte(`{"snapshot_id":"snapshot-1"}`))
	}))
	defer server.Close()

	rec := performSpotifyRequest(t, server.URL, http.MethodDelete, "/v1/playlists/playlist-1/tracks", `{"uris":["spotify:track:one"]}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"snapshot_id":"snapshot-1"`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func performSpotifyRequest(t *testing.T, baseURL, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+handlerTestToken)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	Handler(Config{SpotifyBaseURL: baseURL}).ServeHTTP(rec, req)
	return rec
}

func assertBearer(t *testing.T, r *http.Request) {
	t.Helper()
	if got := r.Header.Get("Authorization"); got != "Bearer "+handlerTestToken {
		t.Fatalf("Authorization = %q", got)
	}
}

func assertErrorCode(t *testing.T, body, want string) {
	t.Helper()
	var got apiErrorResponse
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("decode error response: %v body=%s", err, body)
	}
	if got.Error.Code != want {
		t.Fatalf("error code = %q body=%s", got.Error.Code, body)
	}
}

func serverURL(r *http.Request) string {
	return "http://" + r.Host
}
