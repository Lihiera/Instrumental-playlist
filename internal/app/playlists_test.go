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
			_, _ = w.Write([]byte(`{"items":[{"id":"spotify-id-one","name":"  Focus\tMix\n","external_urls":{"spotify":"https://open.spotify.com/playlist/one"},"owner":{"id":"owner-1"}}],"next":"` + serverURL(r) + `/v1/me/playlists?offset=1"}`))
		case "/v1/me/playlists?offset=1":
			_, _ = w.Write([]byte(`{"items":[{"id":"spotify-id-two","name":"Chill Beats","owner":{"id":"owner-2"}}],"next":null}`))
		default:
			t.Fatalf("unexpected spotify path: %s", r.URL.String())
		}
	}))
	defer server.Close()

	playlistLists := newPlaylistStore()
	router := NewEngine()
	bindSpotifyHandlers(router, Config{SpotifyBaseURL: server.URL}, newTokenStore(), newTrackSearchStore(), playlistLists)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/playlists", nil)
	req.Header.Set("Authorization", "Bearer "+handlerTestToken)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/plain; charset=utf-8") {
		t.Fatalf("Content-Type = %q", got)
	}
	want := "1\tFocus Mix\thttps://open.spotify.com/playlist/one\n2\tChill Beats\t\n"
	if rec.Body.String() != want {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
	for _, leaked := range []string{"spotify-id-one", "spotify-id-two", "owner-1", "owner-2"} {
		if strings.Contains(rec.Body.String(), leaked) {
			t.Fatalf("response leaked raw Spotify field %q: %s", leaked, rec.Body.String())
		}
	}

	first, ok := playlistLists.ByNumber(handlerTestToken, 1)
	if !ok {
		t.Fatal("playlist number 1 was not saved")
	}
	if first.ID != "spotify-id-one" || first.Name != "Focus Mix" || first.URL != "https://open.spotify.com/playlist/one" {
		t.Fatalf("playlist number 1 = %+v", first)
	}
	second, ok := playlistLists.ByNumber(handlerTestToken, 2)
	if !ok {
		t.Fatal("playlist number 2 was not saved")
	}
	if second.ID != "spotify-id-two" || second.Name != "Chill Beats" || second.URL != "" {
		t.Fatalf("playlist number 2 = %+v", second)
	}
}

func TestPlaylistsEndpointUsesStoredUserAccessTokenWhenAuthorizationHeaderIsMissing(t *testing.T) {
	const storedToken = "stored-user-access-token-secret"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer "+storedToken {
			t.Fatalf("Authorization = %q", got)
		}
		_, _ = w.Write([]byte(`{"items":[{"name":"Stored Playlist","external_urls":{"spotify":"https://open.spotify.com/playlist/stored"}}],"next":null}`))
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
	if rec.Body.String() != "1\tStored Playlist\thttps://open.spotify.com/playlist/stored\n" {
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
	bindSpotifyHandlers(router, Config{SpotifyBaseURL: "http://spotify.test"}, tokens, newTrackSearchStore(), newPlaylistStore())

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

func TestCreatePlaylistUsesCurrentSpotifyUserPlaylistAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearer(t, r)
		switch r.URL.Path {
		case "/v1/me/playlists":
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

func TestAddPlaylistTracksUsesCurrentSpotifyItemsAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearer(t, r)
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s", r.Method)
		}
		if r.URL.Path != "/v1/playlists/playlist-1/items" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		var got addTracksRequest
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode spotify request body: %v", err)
		}
		if len(got.URIs) != 1 || got.URIs[0] != "spotify:track:one" || got.Position == nil || *got.Position != 0 {
			t.Fatalf("unexpected body: %+v", got)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"snapshot_id":"snapshot-1"}`))
	}))
	defer server.Close()

	rec := performSpotifyRequest(t, server.URL, http.MethodPost, "/v1/playlists/playlist-1/tracks", `{"uris":["spotify:track:one"],"position":0}`)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"snapshot_id":"snapshot-1"`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestSearchTracksRequiresTerm(t *testing.T) {
	rec := performSpotifyRequest(t, "http://spotify.test", http.MethodGet, "/v1/search/tracks", "")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	assertErrorCode(t, rec.Body.String(), "invalid_request")
}

func TestSearchTracksQueriesInstrumentalAndKaraokeCandidates(t *testing.T) {
	seen := map[string]bool{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearer(t, r)
		if r.URL.Path != "/v1/search" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("type") != "track" || r.URL.Query().Get("limit") != "10" || r.URL.Query().Get("market") != "JP" {
			t.Fatalf("query = %s", r.URL.RawQuery)
		}
		q := r.URL.Query().Get("q")
		seen[q] = true
		switch q {
		case "Original Song instrumental":
			_, _ = w.Write([]byte(`{"tracks":{"items":[{"name":"Original Song - Instrumental","uri":"spotify:track:one","artists":[{"name":"Artist One"}],"album":{"name":"raw album"}}],"next":null}}`))
		case "Original Song カラオケ":
			_, _ = w.Write([]byte(`{"tracks":{"items":[{"name":"Original Song - Karaoke","uri":"spotify:track:two","artists":[{"name":"Artist Two"},{"name":"Artist Three"}],"id":"raw-id"}],"next":null}}`))
		default:
			t.Fatalf("unexpected q = %q", q)
		}
	}))
	defer server.Close()

	searches := newTrackSearchStore()
	router := NewEngine()
	bindSpotifyHandlers(router, Config{SpotifyBaseURL: server.URL}, newTokenStoreWithLatest(handlerTestToken), searches, newPlaylistStore())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/search/tracks?term=Original%20Song", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !seen["Original Song instrumental"] || !seen["Original Song カラオケ"] {
		t.Fatalf("missing expected search queries: %+v", seen)
	}

	var got struct {
		Items []trackSearchItem `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v body=%s", err, rec.Body.String())
	}
	if len(got.Items) != 2 {
		t.Fatalf("items len = %d body=%s", len(got.Items), rec.Body.String())
	}
	if got.Items[0].Name != "Original Song - Instrumental" || got.Items[0].URI != "spotify:track:one" || strings.Join(got.Items[0].Artists, ",") != "Artist One" {
		t.Fatalf("first item = %+v", got.Items[0])
	}
	if got.Items[1].Name != "Original Song - Karaoke" || got.Items[1].URI != "spotify:track:two" || strings.Join(got.Items[1].Artists, ",") != "Artist Two,Artist Three" {
		t.Fatalf("second item = %+v", got.Items[1])
	}
	for _, leaked := range []string{"raw album", "raw-id"} {
		if strings.Contains(rec.Body.String(), leaked) {
			t.Fatalf("response leaked raw Spotify field %q: %s", leaked, rec.Body.String())
		}
	}

	saved, ok := searches.Latest()
	if !ok {
		t.Fatal("latest search was not saved")
	}
	if saved.Term != "Original Song" || len(saved.Items) != 2 {
		t.Fatalf("saved search = %+v", saved)
	}
}

func TestClearTrackSearchCache(t *testing.T) {
	searches := newTrackSearchStore()
	searches.Save("Original Song", []trackSearchItem{{Name: "Original Song - Instrumental", URI: "spotify:track:one"}})

	router := NewEngine()
	bindSpotifyHandlers(router, Config{SpotifyBaseURL: "http://spotify.test"}, newTokenStore(), searches, newPlaylistStore())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/v1/search/tracks/cache", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if _, ok := searches.Latest(); ok {
		t.Fatal("latest search was not cleared")
	}
	if !strings.Contains(rec.Body.String(), `"cleared":true`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/v1/search/tracks/cache", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("second status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"cleared":false`) {
		t.Fatalf("unexpected second body: %s", rec.Body.String())
	}
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
		_, _ = w.Write([]byte(`{"playlists":{"items":[{"id":"playlist-1","name":"Public Focus","external_urls":{"spotify":"https://open.spotify.com/playlist/public"},"tracks":{"total":12}}],"next":null}}`))
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
	if got := searchRec.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/plain; charset=utf-8") {
		t.Fatalf("Content-Type = %q", got)
	}
	if searchRec.Body.String() != "1\tPublic Focus\thttps://open.spotify.com/playlist/public\n" {
		t.Fatalf("unexpected search body: %s", searchRec.Body.String())
	}
	for _, leaked := range []string{"playlist-1", "tracks"} {
		if strings.Contains(searchRec.Body.String(), leaked) {
			t.Fatalf("response leaked raw Spotify field %q: %s", leaked, searchRec.Body.String())
		}
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

func newTokenStoreWithLatest(accessToken string) *tokenStore {
	tokens := newTokenStore()
	if _, err := tokens.Save(storedToken{
		AccessToken: accessToken,
		ExpiresAt:   time.Now().UTC().Add(time.Hour),
	}); err != nil {
		panic(err)
	}
	return tokens
}
