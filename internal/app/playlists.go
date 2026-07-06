package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"instrumental-playlist/internal/instrumental"
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

type conversionRequest struct {
	PlaylistNumber int `json:"playlist_number"`
}

type conversionResponse struct {
	CreatedPlaylist *conversionPlaylistResponse `json:"created_playlist"`
	AddedCount      int                         `json:"added_count"`
	NotFound        []instrumental.Track        `json:"not_found"`
}

type conversionPlaylistResponse struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

type spotifyTrackSearchResponse struct {
	Tracks spotify.Page[spotifyTrackSearchItem] `json:"tracks"`
}

type spotifyArtist struct {
	Name string `json:"name"`
}

type spotifyTrackSearchItem struct {
	Name         string `json:"name"`
	URI          string `json:"uri"`
	ExternalURLs struct {
		Spotify string `json:"spotify"`
	} `json:"external_urls"`
	Artists []spotifyArtist `json:"artists"`
}

type trackSearchItem struct {
	Name    string   `json:"name"`
	URL     string   `json:"url,omitempty"`
	Artists []string `json:"artists"`
	URI     string   `json:"uri"`
}

type spotifyPlaylistTrackItem struct {
	Track        *spotifyPlaylistTrack `json:"track"`
	Item         *spotifyPlaylistTrack `json:"item"`
	Name         string                `json:"name"`
	URI          string                `json:"uri"`
	ExternalURLs struct {
		Spotify string `json:"spotify"`
	} `json:"external_urls"`
	Artists []spotifyArtist `json:"artists"`
}

type spotifyPlaylistTrack struct {
	Name         string `json:"name"`
	URI          string `json:"uri"`
	ExternalURLs struct {
		Spotify string `json:"spotify"`
	} `json:"external_urls"`
	Artists []spotifyArtist `json:"artists"`
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

		body := gin.H{"name": req.Name}
		if req.Description != "" {
			body["description"] = req.Description
		}
		if req.Public != nil {
			body["public"] = *req.Public
		}

		var playlist json.RawMessage
		if err := client.PostJSON(c.Request.Context(), "/v1/me/playlists", opts, body, &playlist); err != nil {
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
		path := spotifyPlaylistItemsPath(strings.TrimSpace(c.Param("playlistID")))
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

		items, err := searchInstrumentalTrackCandidates(c.Request.Context(), client, opts, term)
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

	router.POST("/v1/conversions", func(c *gin.Context) {
		client, opts, ok := spotifyRequest(c, cfg, tokens)
		if !ok {
			return
		}

		var req conversionRequest
		if !bindJSON(c, &req) {
			return
		}
		if req.PlaylistNumber < 1 {
			writeAPIError(c, http.StatusBadRequest, "invalid_request", "playlist_number must be greater than zero", 0)
			return
		}

		source, ok := playlistLists.ByNumber(opts.AccessToken, req.PlaylistNumber)
		if !ok {
			if _, exists := playlistLists.ForAccessToken(opts.AccessToken); exists {
				writeAPIError(c, http.StatusBadRequest, "invalid_request", "playlist_number was not found in the latest playlist list", 0)
				return
			}

			items, err := spotify.GetAllPages[spotifyPlaylistSummary](c.Request.Context(), client, "/v1/me/playlists", opts)
			if err != nil {
				writeSpotifyError(c, err)
				return
			}
			playlistLists.SaveForAccessToken(opts.AccessToken, items)
			writePlaylistLinesStatus(c, http.StatusConflict, items)
			return
		}

		result, err := convertPlaylist(c.Request.Context(), client, opts, source)
		if err != nil {
			var inputErr conversionInputError
			if errors.As(err, &inputErr) {
				writeAPIError(c, http.StatusBadRequest, "invalid_request", inputErr.Error(), 0)
				return
			}
			writeSpotifyError(c, err)
			return
		}
		c.JSON(http.StatusOK, result)
	})
}

func searchInstrumentalTrackCandidates(ctx context.Context, client *spotify.Client, opts spotify.RequestOptions, title string) ([]trackSearchItem, error) {
	var items []trackSearchItem
	for _, suffix := range []string{"instrumental", "カラオケ"} {
		var search spotifyTrackSearchResponse
		query := strings.TrimSpace(title) + " " + suffix
		path := "/v1/search?type=track&limit=10&market=JP&q=" + url.QueryEscape(query)
		if err := client.GetJSON(ctx, path, opts, &search); err != nil {
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
		URL:     strings.TrimSpace(item.ExternalURLs.Spotify),
		Artists: artists,
		URI:     strings.TrimSpace(item.URI),
	}
}

func convertPlaylist(ctx context.Context, client *spotify.Client, opts spotify.RequestOptions, source storedPlaylist) (conversionResponse, error) {
	items, err := spotify.GetAllPages[spotifyPlaylistTrackItem](ctx, client, "/v1/playlists/"+url.PathEscape(source.ID)+"/items", opts)
	if err != nil {
		return conversionResponse{}, spotifyOperationError{Operation: "fetch source playlist tracks", Err: err}
	}
	if len(items) == 0 {
		return conversionResponse{}, conversionInputError("source playlist has no tracks")
	}

	var selectedURIs []string
	notFound := []instrumental.Track{}
	usableTracks := 0
	for _, item := range items {
		original, ok := originalTrackFromPlaylistItem(item)
		if !ok {
			continue
		}
		usableTracks++
		candidates, err := searchInstrumentalTrackCandidates(ctx, client, opts, original.Title)
		if err != nil {
			return conversionResponse{}, spotifyOperationError{Operation: "search instrumental candidates", Err: err}
		}
		selection := instrumental.SelectTarget(original, instrumentalCandidates(candidates))
		if !selection.Found {
			notFound = append(notFound, selection.NotFound)
			continue
		}
		if uri := strings.TrimSpace(selection.Target.URI); uri != "" {
			selectedURIs = append(selectedURIs, uri)
		} else {
			notFound = append(notFound, instrumental.Track{Title: original.Title, URL: original.URL})
		}
	}
	if usableTracks == 0 {
		return conversionResponse{}, conversionInputError("source playlist did not contain playable Spotify tracks")
	}

	if len(selectedURIs) == 0 {
		return conversionResponse{AddedCount: 0, NotFound: notFound}, nil
	}

	created, err := createInstrumentalPlaylist(ctx, client, opts, source.Name)
	if err != nil {
		return conversionResponse{}, spotifyOperationError{Operation: "create destination playlist", Err: err}
	}
	if err := addSpotifyTrackURIs(ctx, client, opts, created.ID, selectedURIs); err != nil {
		return conversionResponse{}, spotifyOperationError{Operation: "add tracks to destination playlist", Err: err}
	}

	return conversionResponse{
		CreatedPlaylist: &conversionPlaylistResponse{
			Title: normalizePlainTextField(created.Name),
			URL:   strings.TrimSpace(created.ExternalURLs.Spotify),
		},
		AddedCount: len(selectedURIs),
		NotFound:   notFound,
	}, nil
}

type conversionInputError string

func (e conversionInputError) Error() string {
	return string(e)
}

type spotifyOperationError struct {
	Operation string
	Err       error
}

func (e spotifyOperationError) Error() string {
	if strings.TrimSpace(e.Operation) == "" {
		return e.Err.Error()
	}
	return e.Operation + " failed: " + e.Err.Error()
}

func (e spotifyOperationError) Unwrap() error {
	return e.Err
}

func originalTrackFromPlaylistItem(item spotifyPlaylistTrackItem) (instrumental.Track, bool) {
	if item.Track != nil {
		return originalTrackFromSpotifyTrack(*item.Track)
	}
	if item.Item != nil {
		return originalTrackFromSpotifyTrack(*item.Item)
	}
	track := instrumental.Track{
		Title:   normalizePlainTextField(item.Name),
		URL:     strings.TrimSpace(item.ExternalURLs.Spotify),
		Artists: spotifyArtistNames(item.Artists),
	}
	return track, track.Title != ""
}

func originalTrackFromSpotifyTrack(track spotifyPlaylistTrack) (instrumental.Track, bool) {
	original := instrumental.Track{
		Title:   normalizePlainTextField(track.Name),
		URL:     strings.TrimSpace(track.ExternalURLs.Spotify),
		Artists: spotifyArtistNames(track.Artists),
	}
	return original, original.Title != ""
}

func instrumentalCandidates(items []trackSearchItem) []instrumental.Candidate {
	candidates := make([]instrumental.Candidate, 0, len(items))
	for _, item := range items {
		candidates = append(candidates, instrumental.Candidate{
			Track: instrumental.Track{
				Title:   normalizePlainTextField(item.Name),
				URL:     strings.TrimSpace(item.URL),
				Artists: append([]string(nil), item.Artists...),
			},
			URI: strings.TrimSpace(item.URI),
		})
	}
	return candidates
}

func createInstrumentalPlaylist(ctx context.Context, client *spotify.Client, opts spotify.RequestOptions, sourceName string) (spotifyPlaylistSummary, error) {
	name := strings.TrimSpace(sourceName)
	if name == "" {
		name = "Instrumental Playlist"
	}
	body := gin.H{"name": name + " Instrumental", "public": false}

	var playlist spotifyPlaylistSummary
	if err := client.PostJSON(ctx, "/v1/me/playlists", opts, body, &playlist); err != nil {
		return spotifyPlaylistSummary{}, err
	}
	if strings.TrimSpace(playlist.ID) == "" {
		return spotifyPlaylistSummary{}, fmt.Errorf("spotify playlist response did not include an id")
	}
	return playlist, nil
}

func addSpotifyTrackURIs(ctx context.Context, client *spotify.Client, opts spotify.RequestOptions, playlistID string, uris []string) error {
	path := spotifyPlaylistItemsPath(playlistID)
	for start := 0; start < len(uris); start += maxSpotifyPlaylistURIs {
		end := start + maxSpotifyPlaylistURIs
		if end > len(uris) {
			end = len(uris)
		}
		if err := client.PostJSON(ctx, path, opts, gin.H{"uris": uris[start:end]}, nil); err != nil {
			return err
		}
	}
	return nil
}

func spotifyPlaylistItemsPath(playlistID string) string {
	return "/v1/playlists/" + url.PathEscape(strings.TrimSpace(playlistID)) + "/items"
}

func spotifyArtistNames(artists []spotifyArtist) []string {
	names := make([]string, 0, len(artists))
	for _, artist := range artists {
		name := strings.TrimSpace(artist.Name)
		if name != "" {
			names = append(names, name)
		}
	}
	return names
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

	var operationErr spotifyOperationError
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
		if errors.As(err, &operationErr) && strings.TrimSpace(operationErr.Operation) != "" {
			message = operationErr.Operation + " failed: " + message
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
	writePlaylistLinesStatus(c, http.StatusOK, playlists)
}

func writePlaylistLinesStatus(c *gin.Context, status int, playlists []spotifyPlaylistSummary) {
	var body strings.Builder
	for i, playlist := range playlists {
		fmt.Fprintf(&body, "%d\t%s\t%s\n", i+1, normalizePlainTextField(playlist.Name), strings.TrimSpace(playlist.ExternalURLs.Spotify))
	}
	c.Data(status, "text/plain; charset=utf-8", []byte(body.String()))
}

func normalizePlainTextField(value string) string {
	fields := strings.Fields(strings.TrimSpace(value))
	return strings.Join(fields, " ")
}
