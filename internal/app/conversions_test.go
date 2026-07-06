package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"instrumental-playlist/internal/spotify"
)

func TestConversionsEndpointRequiresBearerToken(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/conversions", strings.NewReader(`{"playlist_number":1}`))
	req.Header.Set("Content-Type", "application/json")

	Handler(Config{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	assertErrorCode(t, rec.Body.String(), "missing_spotify_access_token")
}

func TestConversionsEndpointFetchesPlaylistChoicesWhenMemoryIsMissing(t *testing.T) {
	var createCalled bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearer(t, r)
		switch r.URL.String() {
		case "/v1/me/playlists":
			_, _ = w.Write([]byte(`{"items":[{"id":"source-playlist-id","name":"  Source\tPlaylist ","external_urls":{"spotify":"https://open.spotify.com/playlist/source"}}],"next":null}`))
		default:
			if r.Method == http.MethodPost {
				createCalled = true
			}
			t.Fatalf("unexpected spotify path: %s", r.URL.String())
		}
	}))
	defer server.Close()

	playlistLists := newPlaylistStore()
	router := NewEngine()
	bindSpotifyHandlers(router, Config{SpotifyBaseURL: server.URL}, newTokenStore(), newTrackSearchStore(), playlistLists)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/conversions", strings.NewReader(`{"playlist_number":1}`))
	req.Header.Set("Authorization", "Bearer "+handlerTestToken)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/plain; charset=utf-8") {
		t.Fatalf("Content-Type = %q", got)
	}
	if rec.Body.String() != "1\tSource Playlist\thttps://open.spotify.com/playlist/source\n" {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
	if createCalled {
		t.Fatal("conversion created a playlist while asking for playlist selection")
	}
	if _, ok := playlistLists.ByNumber(handlerTestToken, 1); !ok {
		t.Fatal("playlist choices were not saved")
	}
	for _, leaked := range []string{"source-playlist-id", handlerTestToken} {
		if strings.Contains(rec.Body.String(), leaked) {
			t.Fatalf("response leaked %q: %s", leaked, rec.Body.String())
		}
	}
}

func TestConversionsEndpointRejectsInvalidPlaylistNumber(t *testing.T) {
	playlistLists := newPlaylistStore()
	playlistLists.SaveForAccessToken(handlerTestToken, []spotifyPlaylistSummary{
		testSpotifyPlaylist("source-playlist-id", "Source Playlist", "https://open.spotify.com/playlist/source"),
	})
	router := NewEngine()
	bindSpotifyHandlers(router, Config{SpotifyBaseURL: "http://spotify.test"}, newTokenStore(), newTrackSearchStore(), playlistLists)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/conversions", strings.NewReader(`{"playlist_number":2}`))
	req.Header.Set("Authorization", "Bearer "+handlerTestToken)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	assertErrorCode(t, rec.Body.String(), "invalid_request")
}

func TestConversionsEndpointCreatesPrivateInstrumentalPlaylist(t *testing.T) {
	var createBody struct {
		Name   string `json:"name"`
		Public *bool  `json:"public"`
	}
	var addBody struct {
		URIs []string `json:"uris"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearer(t, r)
		switch {
		case r.Method == http.MethodGet && r.URL.String() == "/v1/playlists/source-playlist-id/items":
			_, _ = w.Write([]byte(`{"items":[{"track":{"name":"Original Song","uri":"spotify:track:original","external_urls":{"spotify":"https://open.spotify.com/track/original"},"artists":[{"name":"Artist One"}]}},{"track":{"name":"Missing Song","uri":"spotify:track:missing","external_urls":{"spotify":"https://open.spotify.com/track/missing"},"artists":[{"name":"Missing Artist"}]}}],"next":null}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/search":
			switch r.URL.Query().Get("q") {
			case "Original Song instrumental":
				_, _ = w.Write([]byte(`{"tracks":{"items":[{"name":"Original Song - Instrumental","uri":"spotify:track:selected-secret","external_urls":{"spotify":"https://open.spotify.com/track/selected"},"artists":[{"name":"Artist One"}]}],"next":null}}`))
			case "Original Song カラオケ":
				_, _ = w.Write([]byte(`{"tracks":{"items":[{"name":"Original Song カラオケ","uri":"spotify:track:karaoke-secret","external_urls":{"spotify":"https://open.spotify.com/track/karaoke"},"artists":[{"name":"Other Artist"}]}],"next":null}}`))
			case "Missing Song instrumental", "Missing Song カラオケ":
				_, _ = w.Write([]byte(`{"tracks":{"items":[],"next":null}}`))
			default:
				t.Fatalf("unexpected search q = %q", r.URL.Query().Get("q"))
			}
		case r.Method == http.MethodPost && r.URL.Path == "/v1/me/playlists":
			if err := json.NewDecoder(r.Body).Decode(&createBody); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			_, _ = w.Write([]byte(`{"id":"created-playlist-id","name":"Source Playlist Instrumental","external_urls":{"spotify":"https://open.spotify.com/playlist/created"}}`))
		case r.Method == http.MethodPost && r.URL.String() == "/v1/playlists/created-playlist-id/items":
			if err := json.NewDecoder(r.Body).Decode(&addBody); err != nil {
				t.Fatalf("decode add body: %v", err)
			}
			_, _ = w.Write([]byte(`{"snapshot_id":"snapshot-1"}`))
		default:
			t.Fatalf("unexpected spotify request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	playlistLists := newPlaylistStore()
	playlistLists.SaveForAccessToken(handlerTestToken, []spotifyPlaylistSummary{
		testSpotifyPlaylist("source-playlist-id", "Source Playlist", "https://open.spotify.com/playlist/source"),
	})
	router := NewEngine()
	bindSpotifyHandlers(router, Config{SpotifyBaseURL: server.URL}, newTokenStore(), newTrackSearchStore(), playlistLists)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/conversions", strings.NewReader(`{"playlist_number":1}`))
	req.Header.Set("Authorization", "Bearer "+handlerTestToken)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if createBody.Name != "Source Playlist Instrumental" || createBody.Public == nil || *createBody.Public {
		t.Fatalf("create body = %+v", createBody)
	}
	if len(addBody.URIs) != 1 || addBody.URIs[0] != "spotify:track:selected-secret" {
		t.Fatalf("add body = %+v", addBody)
	}

	var got conversionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v body=%s", err, rec.Body.String())
	}
	if got.CreatedPlaylist == nil || got.CreatedPlaylist.Title != "Source Playlist Instrumental" || got.CreatedPlaylist.URL != "https://open.spotify.com/playlist/created" {
		t.Fatalf("created playlist = %+v", got.CreatedPlaylist)
	}
	if got.AddedCount != 1 {
		t.Fatalf("added_count = %d", got.AddedCount)
	}
	if len(got.NotFound) != 1 || got.NotFound[0].Title != "Missing Song" || got.NotFound[0].URL != "https://open.spotify.com/track/missing" {
		t.Fatalf("not_found = %+v", got.NotFound)
	}
	for _, leaked := range []string{"spotify:track:selected-secret", "spotify:track:karaoke-secret", "source-playlist-id", handlerTestToken} {
		if strings.Contains(rec.Body.String(), leaked) {
			t.Fatalf("response leaked %q: %s", leaked, rec.Body.String())
		}
	}
}

func TestConversionsEndpointDoesNotUseKaraokeSearchFallbackWhenNameOmitsKaraoke(t *testing.T) {
	var writeCalled bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearer(t, r)
		switch {
		case r.Method == http.MethodGet && r.URL.String() == "/v1/playlists/source-playlist-id/items":
			_, _ = w.Write([]byte(`{"items":[{"item":{"name":"アンコール","external_urls":{"spotify":"https://open.spotify.com/track/original"},"artists":[{"name":"YOASOBI"}]}}],"next":null}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/search":
			switch r.URL.Query().Get("q") {
			case "アンコール instrumental":
				_, _ = w.Write([]byte(`{"tracks":{"items":[],"next":null}}`))
			case "アンコール カラオケ":
				_, _ = w.Write([]byte(`{"tracks":{"items":[{"name":"アンコール","uri":"spotify:track:karaoke-selected","artists":[{"name":"Karaoke Artist"}]}],"next":null}}`))
			default:
				t.Fatalf("unexpected search q = %q", r.URL.Query().Get("q"))
			}
		case r.Method == http.MethodPost:
			writeCalled = true
			t.Fatalf("unexpected spotify write: %s", r.URL.String())
		default:
			t.Fatalf("unexpected spotify request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	playlistLists := newPlaylistStore()
	playlistLists.SaveForAccessToken(handlerTestToken, []spotifyPlaylistSummary{
		testSpotifyPlaylist("source-playlist-id", "Source Playlist", "https://open.spotify.com/playlist/source"),
	})
	router := NewEngine()
	bindSpotifyHandlers(router, Config{SpotifyBaseURL: server.URL}, newTokenStore(), newTrackSearchStore(), playlistLists)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/conversions", strings.NewReader(`{"playlist_number":1}`))
	req.Header.Set("Authorization", "Bearer "+handlerTestToken)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if writeCalled {
		t.Fatal("conversion wrote playlist for karaoke search result without keyword")
	}
	var got conversionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v body=%s", err, rec.Body.String())
	}
	if got.CreatedPlaylist != nil || got.AddedCount != 0 || len(got.NotFound) != 1 || got.NotFound[0].Title != "アンコール" {
		t.Fatalf("response = %+v", got)
	}
}

func TestConversionsEndpointReadsDirectTrackItems(t *testing.T) {
	var addBody struct {
		URIs []string `json:"uris"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearer(t, r)
		switch {
		case r.Method == http.MethodGet && r.URL.String() == "/v1/playlists/source-playlist-id/items":
			_, _ = w.Write([]byte(`{"items":[{"name":"Direct Song","external_urls":{"spotify":"https://open.spotify.com/track/direct"},"artists":[{"name":"Artist One"}]}],"next":null}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/search":
			switch r.URL.Query().Get("q") {
			case "Direct Song instrumental":
				_, _ = w.Write([]byte(`{"tracks":{"items":[{"name":"Direct Song - Instrumental","uri":"spotify:track:direct-selected","artists":[{"name":"Artist One"}]}],"next":null}}`))
			case "Direct Song カラオケ":
				_, _ = w.Write([]byte(`{"tracks":{"items":[],"next":null}}`))
			default:
				t.Fatalf("unexpected search q = %q", r.URL.Query().Get("q"))
			}
		case r.Method == http.MethodPost && r.URL.String() == "/v1/me/playlists":
			_, _ = w.Write([]byte(`{"id":"created-playlist-id","name":"Source Playlist Instrumental","external_urls":{"spotify":"https://open.spotify.com/playlist/created"}}`))
		case r.Method == http.MethodPost && r.URL.String() == "/v1/playlists/created-playlist-id/items":
			if err := json.NewDecoder(r.Body).Decode(&addBody); err != nil {
				t.Fatalf("decode add body: %v", err)
			}
			_, _ = w.Write([]byte(`{"snapshot_id":"snapshot-1"}`))
		default:
			t.Fatalf("unexpected spotify request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	playlistLists := newPlaylistStore()
	playlistLists.SaveForAccessToken(handlerTestToken, []spotifyPlaylistSummary{
		testSpotifyPlaylist("source-playlist-id", "Source Playlist", "https://open.spotify.com/playlist/source"),
	})
	router := NewEngine()
	bindSpotifyHandlers(router, Config{SpotifyBaseURL: server.URL}, newTokenStore(), newTrackSearchStore(), playlistLists)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/conversions", strings.NewReader(`{"playlist_number":1}`))
	req.Header.Set("Authorization", "Bearer "+handlerTestToken)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if len(addBody.URIs) != 1 || addBody.URIs[0] != "spotify:track:direct-selected" {
		t.Fatalf("add body = %+v", addBody)
	}
}

func TestConversionsEndpointReadsItemTrackPayloads(t *testing.T) {
	var addBody struct {
		URIs []string `json:"uris"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearer(t, r)
		switch {
		case r.Method == http.MethodGet && r.URL.String() == "/v1/playlists/source-playlist-id/items":
			_, _ = w.Write([]byte(`{"items":[{"added_at":"2026-07-05T18:11:51Z","item":{"type":"track","track":true,"name":"優しい彗星","uri":"spotify:track:original","external_urls":{"spotify":"https://open.spotify.com/track/original"},"artists":[{"name":"YOASOBI"}]}}],"next":null}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/search":
			switch r.URL.Query().Get("q") {
			case "優しい彗星 instrumental":
				_, _ = w.Write([]byte(`{"tracks":{"items":[{"name":"優しい彗星 Instrumental","uri":"spotify:track:selected","artists":[{"name":"YOASOBI"}]}],"next":null}}`))
			case "優しい彗星 カラオケ":
				_, _ = w.Write([]byte(`{"tracks":{"items":[],"next":null}}`))
			default:
				t.Fatalf("unexpected search q = %q", r.URL.Query().Get("q"))
			}
		case r.Method == http.MethodPost && r.URL.String() == "/v1/me/playlists":
			_, _ = w.Write([]byte(`{"id":"created-playlist-id","name":"Source Playlist Instrumental","external_urls":{"spotify":"https://open.spotify.com/playlist/created"}}`))
		case r.Method == http.MethodPost && r.URL.String() == "/v1/playlists/created-playlist-id/items":
			if err := json.NewDecoder(r.Body).Decode(&addBody); err != nil {
				t.Fatalf("decode add body: %v", err)
			}
			_, _ = w.Write([]byte(`{"snapshot_id":"snapshot-1"}`))
		default:
			t.Fatalf("unexpected spotify request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	playlistLists := newPlaylistStore()
	playlistLists.SaveForAccessToken(handlerTestToken, []spotifyPlaylistSummary{
		testSpotifyPlaylist("source-playlist-id", "Source Playlist", "https://open.spotify.com/playlist/source"),
	})
	router := NewEngine()
	bindSpotifyHandlers(router, Config{SpotifyBaseURL: server.URL}, newTokenStore(), newTrackSearchStore(), playlistLists)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/conversions", strings.NewReader(`{"playlist_number":1}`))
	req.Header.Set("Authorization", "Bearer "+handlerTestToken)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if len(addBody.URIs) != 1 || addBody.URIs[0] != "spotify:track:selected" {
		t.Fatalf("add body = %+v", addBody)
	}
}

func TestConversionsEndpointReportsSpotifyOperationForForbiddenErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearer(t, r)
		switch {
		case r.Method == http.MethodGet && r.URL.String() == "/v1/playlists/source-playlist-id/items":
			_, _ = w.Write([]byte(`{"items":[{"item":{"name":"Original Song","external_urls":{"spotify":"https://open.spotify.com/track/original"},"artists":[{"name":"Artist One"}]}}],"next":null}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/search":
			switch r.URL.Query().Get("q") {
			case "Original Song instrumental":
				_, _ = w.Write([]byte(`{"tracks":{"items":[{"name":"Original Song Instrumental","uri":"spotify:track:selected","artists":[{"name":"Artist One"}]}],"next":null}}`))
			case "Original Song カラオケ":
				_, _ = w.Write([]byte(`{"tracks":{"items":[],"next":null}}`))
			default:
				t.Fatalf("unexpected search q = %q", r.URL.Query().Get("q"))
			}
		case r.Method == http.MethodPost && r.URL.String() == "/v1/me/playlists":
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":{"status":403,"message":"Forbidden"}}`))
		default:
			t.Fatalf("unexpected spotify request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	playlistLists := newPlaylistStore()
	playlistLists.SaveForAccessToken(handlerTestToken, []spotifyPlaylistSummary{
		testSpotifyPlaylist("source-playlist-id", "Source Playlist", "https://open.spotify.com/playlist/source"),
	})
	router := NewEngine()
	bindSpotifyHandlers(router, Config{SpotifyBaseURL: server.URL}, newTokenStore(), newTrackSearchStore(), playlistLists)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/conversions", strings.NewReader(`{"playlist_number":1}`))
	req.Header.Set("Authorization", "Bearer "+handlerTestToken)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	assertErrorCode(t, rec.Body.String(), "spotify_api_error")
	if !strings.Contains(rec.Body.String(), "create destination playlist failed: Forbidden") {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), handlerTestToken) || strings.Contains(rec.Body.String(), "source-playlist-id") {
		t.Fatalf("response leaked internal value: %s", rec.Body.String())
	}
}

func TestConversionsEndpointDoesNotCreateEmptyPlaylist(t *testing.T) {
	var createCalled bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearer(t, r)
		switch {
		case r.Method == http.MethodGet && r.URL.String() == "/v1/playlists/source-playlist-id/items":
			_, _ = w.Write([]byte(`{"items":[{"track":{"name":"Missing Song","external_urls":{"spotify":"https://open.spotify.com/track/missing"},"artists":[{"name":"Missing Artist"}]}}],"next":null}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/search":
			_, _ = w.Write([]byte(`{"tracks":{"items":[],"next":null}}`))
		case r.Method == http.MethodPost:
			createCalled = true
			t.Fatalf("unexpected playlist write: %s", r.URL.String())
		default:
			t.Fatalf("unexpected spotify request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	playlistLists := newPlaylistStore()
	playlistLists.SaveForAccessToken(handlerTestToken, []spotifyPlaylistSummary{
		testSpotifyPlaylist("source-playlist-id", "Source Playlist", "https://open.spotify.com/playlist/source"),
	})
	router := NewEngine()
	bindSpotifyHandlers(router, Config{SpotifyBaseURL: server.URL}, newTokenStore(), newTrackSearchStore(), playlistLists)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/conversions", strings.NewReader(`{"playlist_number":1}`))
	req.Header.Set("Authorization", "Bearer "+handlerTestToken)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if createCalled {
		t.Fatal("created playlist for all-not-found conversion")
	}
	var got conversionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v body=%s", err, rec.Body.String())
	}
	if got.CreatedPlaylist != nil || got.AddedCount != 0 || len(got.NotFound) != 1 {
		t.Fatalf("response = %+v", got)
	}
}

func TestConversionsEndpointRejectsSourcePlaylistWithNoTracks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearer(t, r)
		if r.Method != http.MethodGet || r.URL.String() != "/v1/playlists/source-playlist-id/items" {
			t.Fatalf("unexpected spotify request: %s %s", r.Method, r.URL.String())
		}
		_, _ = w.Write([]byte(`{"items":[],"next":null}`))
	}))
	defer server.Close()

	playlistLists := newPlaylistStore()
	playlistLists.SaveForAccessToken(handlerTestToken, []spotifyPlaylistSummary{
		testSpotifyPlaylist("source-playlist-id", "Source Playlist", "https://open.spotify.com/playlist/source"),
	})
	router := NewEngine()
	bindSpotifyHandlers(router, Config{SpotifyBaseURL: server.URL}, newTokenStore(), newTrackSearchStore(), playlistLists)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/conversions", strings.NewReader(`{"playlist_number":1}`))
	req.Header.Set("Authorization", "Bearer "+handlerTestToken)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	assertErrorCode(t, rec.Body.String(), "invalid_request")
	if !strings.Contains(rec.Body.String(), "source playlist has no tracks") {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestConversionsEndpointRejectsSourcePlaylistWithNoPlayableSpotifyTracks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearer(t, r)
		if r.Method != http.MethodGet || r.URL.String() != "/v1/playlists/source-playlist-id/items" {
			t.Fatalf("unexpected spotify request: %s %s", r.Method, r.URL.String())
		}
		_, _ = w.Write([]byte(`{"items":[{"track":null},{"track":{"name":"   "}}],"next":null}`))
	}))
	defer server.Close()

	playlistLists := newPlaylistStore()
	playlistLists.SaveForAccessToken(handlerTestToken, []spotifyPlaylistSummary{
		testSpotifyPlaylist("source-playlist-id", "Source Playlist", "https://open.spotify.com/playlist/source"),
	})
	router := NewEngine()
	bindSpotifyHandlers(router, Config{SpotifyBaseURL: server.URL}, newTokenStore(), newTrackSearchStore(), playlistLists)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/conversions", strings.NewReader(`{"playlist_number":1}`))
	req.Header.Set("Authorization", "Bearer "+handlerTestToken)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	assertErrorCode(t, rec.Body.String(), "invalid_request")
	if !strings.Contains(rec.Body.String(), "source playlist did not contain playable Spotify tracks") {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestAddSpotifyTrackURIsBatchesSpotifyRequests(t *testing.T) {
	var batchSizes []int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearer(t, r)
		if r.Method != http.MethodPost || r.URL.String() != "/v1/playlists/created-playlist-id/items" {
			t.Fatalf("unexpected spotify request: %s %s", r.Method, r.URL.String())
		}
		var got struct {
			URIs []string `json:"uris"`
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode add body: %v", err)
		}
		batchSizes = append(batchSizes, len(got.URIs))
		_, _ = w.Write([]byte(`{"snapshot_id":"snapshot-1"}`))
	}))
	defer server.Close()

	client, err := spotify.New(spotify.Config{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("spotify.New returned error: %v", err)
	}
	uris := make([]string, maxSpotifyPlaylistURIs+1)
	for i := range uris {
		uris[i] = "spotify:track:test"
	}

	if err := addSpotifyTrackURIs(context.Background(), client, spotify.RequestOptions{AccessToken: handlerTestToken}, "created-playlist-id", uris); err != nil {
		t.Fatalf("addSpotifyTrackURIs returned error: %v", err)
	}
	if len(batchSizes) != 2 || batchSizes[0] != maxSpotifyPlaylistURIs || batchSizes[1] != 1 {
		t.Fatalf("batch sizes = %+v", batchSizes)
	}
}

func testSpotifyPlaylist(id, name, spotifyURL string) spotifyPlaylistSummary {
	var playlist spotifyPlaylistSummary
	playlist.ID = id
	playlist.Name = name
	playlist.ExternalURLs.Spotify = spotifyURL
	return playlist
}
