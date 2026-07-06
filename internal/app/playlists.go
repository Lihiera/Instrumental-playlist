package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"instrumental-playlist/internal/spotify"
)

const maxSpotifyPlaylistURIs = 100

type apiErrorResponse struct {
	Error apiErrorBody `json:"error"`
}

type apiErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"status,omitempty"`
}

type createPlaylistRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Public      *bool  `json:"public,omitempty"`
}

type addTracksRequest struct {
	URIs     []string `json:"uris"`
	Position *int     `json:"position,omitempty"`
}

type removeTracksRequest struct {
	URIs       []string `json:"uris"`
	SnapshotID string   `json:"snapshot_id,omitempty"`
}

type spotifyUser struct {
	ID string `json:"id"`
}

type spotifyTrackSearchResponse struct {
	Tracks spotify.Page[spotifyTrackSearchItem] `json:"tracks"`
}

type spotifyTrackSearchItem struct {
	Name    string `json:"name"`
	URI     string `json:"uri"`
	Artists []struct {
		Name string `json:"name"`
	} `json:"artists"`
}

type trackSearchItem struct {
	Name    string   `json:"name"`
	Artists []string `json:"artists"`
	URI     string   `json:"uri"`
}

type spotifyPlaylistSearchResponse struct {
	Playlists spotify.Page[spotifyPlaylistSummary] `json:"playlists"`
}

type spotifyPlaylistSummary struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	ExternalURLs struct {
		Spotify string `json:"spotify"`
	} `json:"external_urls"`
}

func bindSpotifyHandlers(router *gin.Engine, cfg Config, tokens *tokenStore, trackSearches *trackSearchStore, playlistLists *playlistStore) {
	router.GET("/v1/playlists", func(c *gin.Context) {
		client, opts, ok := spotifyRequest(c, cfg, tokens)
		if !ok {
			return
		}

		items, err := spotify.GetAllPages[spotifyPlaylistSummary](c.Request.Context(), client, "/v1/me/playlists", opts)
		if err != nil {
			writeSpotifyError(c, err)
			return
		}
		playlistLists.SaveForAccessToken(opts.AccessToken, items)
		writePlaylistLines(c, items)
	})

	router.POST("/v1/playlists", func(c *gin.Context) {
		client, opts, ok := spotifyRequest(c, cfg, tokens)
		if !ok {
			return
		}

		var req createPlaylistRequest
		if !bindJSON(c, &req) {
			return
		}
		req.Name = strings.TrimSpace(req.Name)
		if req.Name == "" {
			writeAPIError(c, http.StatusBadRequest, "invalid_request", "name is required", 0)
			return
		}

		var me spotifyUser
		if err := client.GetJSON(c.Request.Context(), "/v1/me", opts, &me); err != nil {
			writeSpotifyError(c, err)
			return
		}
		if strings.TrimSpace(me.ID) == "" {
			writeAPIError(c, http.StatusBadGateway, "spotify_request_failed", "spotify user response did not include an id", 0)
			return
		}

		body := gin.H{"name": req.Name}
		if req.Description != "" {
			body["description"] = req.Description
		}
		if req.Public != nil {
			body["public"] = *req.Public
		}

		var playlist json.RawMessage
		if err := client.PostJSON(c.Request.Context(), "/v1/users/"+url.PathEscape(me.ID)+"/playlists", opts, body, &playlist); err != nil {
			writeSpotifyError(c, err)
			return
		}
		writeRawJSON(c, http.StatusCreated, playlist)
	})

	router.GET("/v1/playlists/:playlistID/tracks", func(c *gin.Context) {
		client, opts, ok := spotifyRequest(c, cfg, tokens)
		if !ok {
			return
		}

		playlistID := strings.TrimSpace(c.Param("playlistID"))
		items, err := spotify.GetAllPages[json.RawMessage](c.Request.Context(), client, "/v1/playlists/"+url.PathEscape(playlistID)+"/items", opts)
		if err != nil {
			writeSpotifyError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": items})
	})

	router.POST("/v1/playlists/:playlistID/tracks", func(c *gin.Context) {
		client, opts, ok := spotifyRequest(c, cfg, tokens)
		if !ok {
			return
		}

		var req addTracksRequest
		if !bindJSON(c, &req) {
			return
		}
		uris, ok := validateURIList(c, req.URIs)
		if !ok {
			return
		}

		body := gin.H{"uris": uris}
		if req.Position != nil {
			body["position"] = *req.Position
		}

		var result json.RawMessage
		path := "/v1/playlists/" + url.PathEscape(strings.TrimSpace(c.Param("playlistID"))) + "/tracks"
		if err := client.PostJSON(c.Request.Context(), path, opts, body, &result); err != nil {
			writeSpotifyError(c, err)
			return
		}
		writeRawJSON(c, http.StatusCreated, result)
	})

	router.DELETE("/v1/playlists/:playlistID/tracks", func(c *gin.Context) {
		client, opts, ok := spotifyRequest(c, cfg, tokens)
		if !ok {
			return
		}

		var req removeTracksRequest
		if !bindJSON(c, &req) {
			return
		}
		uris, ok := validateURIList(c, req.URIs)
		if !ok {
			return
		}

		tracks := make([]gin.H, 0, len(uris))
		for _, uri := range uris {
			tracks = append(tracks, gin.H{"uri": uri})
		}
		body := gin.H{"tracks": tracks}
		if strings.TrimSpace(req.SnapshotID) != "" {
			body["snapshot_id"] = strings.TrimSpace(req.SnapshotID)
		}

		var result json.RawMessage
		path := "/v1/playlists/" + url.PathEscape(strings.TrimSpace(c.Param("playlistID"))) + "/tracks"
		if err := client.DeleteJSON(c.Request.Context(), path, opts, body, &result); err != nil {
			writeSpotifyError(c, err)
			return
		}
		writeRawJSON(c, http.StatusOK, result)
	})

	router.GET("/v1/search/tracks", func(c *gin.Context) {
		client, opts, ok := spotifyRequest(c, cfg, tokens)
		if !ok {
			return
		}

		term := strings.TrimSpace(c.Query("term"))
		if term == "" {
			writeAPIError(c, http.StatusBadRequest, "invalid_request", "term is required", 0)
			return
		}

		items, err := searchInstrumentalTrackCandidates(c, client, opts, term)
		if err != nil {
			writeSpotifyError(c, err)
			return
		}
		trackSearches.Save(term, items)
		c.JSON(http.StatusOK, gin.H{"items": items})
	})

	router.GET("/v1/noLogin/search/playlists", func(c *gin.Context) {
		keyword := strings.TrimSpace(c.Query("keyword"))
		if keyword == "" {
			writeAPIError(c, http.StatusBadRequest, "invalid_request", "keyword is required", 0)
			return
		}

		client, err := cfg.SpotifyClient()
		if err != nil {
			writeAPIError(c, http.StatusInternalServerError, "spotify_client_config_invalid", "spotify client configuration is invalid", 0)
			return
		}
		opts, ok := serverAppOnlySpotifyOptions(c, cfg)
		if !ok {
			return
		}

		var search spotifyPlaylistSearchResponse
		path := "/v1/search?type=playlist&limit=10&market=JP&q=" + url.QueryEscape(keyword)
		if err := client.GetJSON(c.Request.Context(), path, opts, &search); err != nil {
			writeSpotifyError(c, err)
			return
		}
		writePlaylistLines(c, search.Playlists.Items)
	})
}

func searchInstrumentalTrackCandidates(c *gin.Context, client *spotify.Client, opts spotify.RequestOptions, title string) ([]trackSearchItem, error) {
	var items []trackSearchItem
	for _, suffix := range []string{"instrumental", "カラオケ"} {
		var search spotifyTrackSearchResponse
		query := strings.TrimSpace(title) + " " + suffix
		path := "/v1/search?type=track&limit=10&market=JP&q=" + url.QueryEscape(query)
		if err := client.GetJSON(c.Request.Context(), path, opts, &search); err != nil {
			return nil, err
		}
		for _, item := range search.Tracks.Items {
			items = append(items, simplifyTrackSearchItem(item))
		}
	}
	return items, nil
}

func simplifyTrackSearchItem(item spotifyTrackSearchItem) trackSearchItem {
	artists := make([]string, 0, len(item.Artists))
	for _, artist := range item.Artists {
		name := strings.TrimSpace(artist.Name)
		if name != "" {
			artists = append(artists, name)
		}
	}
	return trackSearchItem{
		Name:    strings.TrimSpace(item.Name),
		Artists: artists,
		URI:     strings.TrimSpace(item.URI),
	}
}

func spotifyRequest(c *gin.Context, cfg Config, tokens *tokenStore) (*spotify.Client, spotify.RequestOptions, bool) {
	token, ok := userAccessToken(c, tokens)
	if !ok {
		return nil, spotify.RequestOptions{}, false
	}

	client, err := cfg.SpotifyClient()
	if err != nil {
		writeAPIError(c, http.StatusInternalServerError, "spotify_client_config_invalid", "spotify client configuration is invalid", 0)
		return nil, spotify.RequestOptions{}, false
	}

	return client, spotify.RequestOptions{AccessToken: token}, true
}

func userAccessToken(c *gin.Context, tokens *tokenStore) (string, bool) {
	header := strings.TrimSpace(c.GetHeader("Authorization"))
	if header != "" {
		token, err := bearerToken(header)
		if err != nil {
			writeSpotifyError(c, err)
			return "", false
		}
		return token, true
	}

	stored, ok := tokens.Latest()
	if !ok || strings.TrimSpace(stored.AccessToken) == "" {
		writeSpotifyError(c, spotify.ErrMissingAccessToken)
		return "", false
	}
	if accessTokenExpired(stored, time.Now().UTC()) {
		writeAPIError(c, http.StatusUnauthorized, "spotify_access_token_expired", "stored Spotify access token is expired; login again", 0)
		return "", false
	}

	return strings.TrimSpace(stored.AccessToken), true
}

func bearerToken(header string) (string, error) {
	kind, token, ok := strings.Cut(strings.TrimSpace(header), " ")
	if !ok || !strings.EqualFold(kind, "Bearer") || strings.TrimSpace(token) == "" {
		return "", spotify.ErrMissingAccessToken
	}
	return strings.TrimSpace(token), nil
}

func serverAppOnlySpotifyOptions(c *gin.Context, cfg Config) (spotify.RequestOptions, bool) {
	client, err := cfg.SpotifyAuthClient()
	if err != nil {
		writeAuthConfigError(c, err)
		return spotify.RequestOptions{}, false
	}
	token, err := client.ClientCredentialsToken(c.Request.Context())
	if err != nil {
		writeAuthError(c, err)
		return spotify.RequestOptions{}, false
	}
	return spotify.RequestOptions{AccessToken: token.AccessToken}, true
}

func bindJSON(c *gin.Context, out any) bool {
	if err := c.ShouldBindJSON(out); err != nil {
		writeAPIError(c, http.StatusBadRequest, "invalid_json", "request body must be valid JSON", 0)
		return false
	}
	return true
}

func validateURIList(c *gin.Context, raw []string) ([]string, bool) {
	if len(raw) == 0 {
		writeAPIError(c, http.StatusBadRequest, "invalid_request", "uris must contain at least one Spotify URI", 0)
		return nil, false
	}
	if len(raw) > maxSpotifyPlaylistURIs {
		writeAPIError(c, http.StatusBadRequest, "invalid_request", "uris must contain at most 100 Spotify URIs", 0)
		return nil, false
	}

	uris := make([]string, 0, len(raw))
	for _, value := range raw {
		uri := strings.TrimSpace(value)
		if uri == "" {
			writeAPIError(c, http.StatusBadRequest, "invalid_request", "uris cannot contain empty values", 0)
			return nil, false
		}
		uris = append(uris, uri)
	}
	return uris, true
}

func writeSpotifyError(c *gin.Context, err error) {
	if errors.Is(err, spotify.ErrMissingAccessToken) {
		writeAPIError(c, http.StatusUnauthorized, "missing_spotify_access_token", "Authorization bearer token is required", 0)
		return
	}

	var apiErr *spotify.APIError
	if errors.As(err, &apiErr) {
		status := apiErr.StatusCode
		if status < http.StatusBadRequest || status >= 600 {
			status = http.StatusBadGateway
		}
		message := apiErr.SpotifyError.Message
		if strings.TrimSpace(message) == "" {
			message = "spotify api request failed"
		}
		writeAPIError(c, status, "spotify_api_error", message, apiErr.StatusCode)
		return
	}

	writeAPIError(c, http.StatusBadGateway, "spotify_request_failed", "spotify request failed", 0)
}

func writeAPIError(c *gin.Context, httpStatus int, code, message string, upstreamStatus int) {
	c.JSON(httpStatus, apiErrorResponse{
		Error: apiErrorBody{
			Code:    code,
			Message: message,
			Status:  upstreamStatus,
		},
	})
}

func writeRawJSON(c *gin.Context, status int, body json.RawMessage) {
	if len(body) == 0 {
		c.JSON(status, gin.H{})
		return
	}
	c.Data(status, "application/json; charset=utf-8", body)
}

func writePlaylistLines(c *gin.Context, playlists []spotifyPlaylistSummary) {
	var body strings.Builder
	for i, playlist := range playlists {
		fmt.Fprintf(&body, "%d\t%s\t%s\n", i+1, normalizePlainTextField(playlist.Name), strings.TrimSpace(playlist.ExternalURLs.Spotify))
	}
	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(body.String()))
}

func normalizePlainTextField(value string) string {
	fields := strings.Fields(strings.TrimSpace(value))
	return strings.Join(fields, " ")
}
