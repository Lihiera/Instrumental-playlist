# API Reference

## Runtime Settings

The server reads `.env` first, then process environment variables. Process environment variables take precedence.

Required or supported settings:

- `HTTP_ADDR`: HTTP listen address. Default: `:8080`.
- `SPOTIFY_CLIENT_ID`: Spotify application Client ID.
- `SPOTIFY_CLIENT_SECRET`: Spotify application Client Secret. This value is never returned by API responses.
- `SPOTIFY_REDIRECT_URI`: Redirect URI registered in the Spotify developer dashboard.
- `SPOTIFY_BASE_URL`: Spotify Web API base URL. Default: `https://api.spotify.com`.
- `SPOTIFY_ACCOUNTS_BASE_URL`: Spotify Accounts base URL. Default: `https://accounts.spotify.com`.
## Endpoints

### `GET /health`

Returns a lightweight health response.

```json
{"status":"ok"}
```

### `GET /v1/config`

Returns public runtime configuration. Secret values are redacted by omission.

```json
{
  "http_addr": ":8080",
  "spotify_client_id_configured": true,
  "spotify_client_secret_configured": true,
  "spotify_redirect_uri": "http://localhost:8080/auth/spotify/callback",
  "spotify_base_url": "https://api.spotify.com",
  "spotify_accounts_base_url": "https://accounts.spotify.com"
}
```

### `POST /v1/auth/tokens`

Stores a Spotify access token and optional refresh token in process memory. Tokens are not returned by this endpoint.

Request:

```json
{
  "access_token": "spotify-user-access-token",
  "refresh_token": "optional-refresh-token",
  "token_type": "bearer",
  "scope": "playlist-read-private playlist-modify-private",
  "expires_in": 3600
}
```

Response:

```json
{
  "id": "token-metadata-id",
  "token_type": "bearer",
  "scope": "playlist-read-private playlist-modify-private",
  "expires_at": "2026-07-06T12:00:00Z",
  "has_refresh_token": true
}
```

### `GET /v1/auth/tokens/{tokenID}`

Returns metadata for a token stored in process memory. Access tokens and refresh tokens are not returned.

```json
{
  "id": "token-metadata-id",
  "token_type": "bearer",
  "scope": "playlist-read-private playlist-modify-private",
  "expires_at": "2026-07-06T12:00:00Z",
  "has_refresh_token": true
}
```

### `GET /v1/auth/status`

Returns whether a Spotify user token is currently stored in process memory. Token values are not returned.

Logged-out response:

```json
{
  "logged_in": false
}
```

Logged-in response:

```json
{
  "logged_in": true,
  "token": {
    "id": "token-metadata-id",
    "token_type": "Bearer",
    "scope": "playlist-read-private playlist-modify-private playlist-modify-public",
    "expires_at": "2026-07-06T12:00:00Z",
    "has_refresh_token": true
  },
  "access_token_expired": false
}
```

### `POST /v1/auth/logout`

Clears all Spotify user tokens from process memory. The endpoint is idempotent and does not return token values.

Response when a token was cleared:

```json
{
  "logged_out": true
}
```

Response when no token was stored:

```json
{
  "logged_out": false
}
```

## Authorization Code Flow Endpoints

### `GET /oauth/spotify/login`

Creates a one-time OAuth `state` value in process memory and redirects the user to Spotify Accounts authorization. The authorization request includes playlist scopes needed by this API:

- `playlist-read-private`
- `playlist-modify-public`
- `playlist-modify-private`

### `GET /oauth/spotify/callback`

Validates `state`, exchanges Spotify's authorization `code` for access and refresh tokens, and stores token metadata in process memory. Token values are not returned.

Successful response: `200 text/html` success page for browser login flows. The page displays only safe token metadata:

- token metadata id
- token type
- granted scopes
- expiration time
- whether a refresh token was saved

Redis-backed callback state and token storage is deferred until after the core feature set is complete.

## Spotify Request Headers

Spotify playlist and user track search endpoints use a Spotify user access token. After a successful `/oauth/spotify/login` flow, the server uses the latest user token stored in process memory automatically.

Clients may also pass a token explicitly through the standard `Authorization` header. An explicit header takes precedence over the stored in-memory token.

```http
Authorization: Bearer replace-with-spotify-access-token
```

The access token must include the scopes required by the operation. Playlist operations need `playlist-read-private`, `playlist-modify-public`, and/or `playlist-modify-private`. If no stored token or explicit bearer token is available, these endpoints return `401 missing_spotify_access_token`. If the stored access token has expired, they return `401 spotify_access_token_expired`.

## Spotify Endpoints

### `GET /v1/playlists`

Returns the current Spotify user's playlists as plain text. The server follows Spotify pagination, then returns one playlist per line with only number, name, and Spotify URL separated by tabs.

```text
1	Instrumental Focus	https://open.spotify.com/playlist/example
2	Study Beats	https://open.spotify.com/playlist/example-two
```

The response number is the playlist selector for later instrumental conversion requests. Each `GET /v1/playlists` call replaces the latest in-memory playlist list for that authenticated user only. The saved list keeps the Spotify playlist id internally, but playlist ids are not returned in this response.

An empty playlist result returns `200 text/plain` with an empty body.

### `POST /v1/playlists`

Creates a playlist for the current Spotify user through Spotify's `POST /v1/me/playlists` endpoint.

Request:

```json
{
  "name": "Instrumental Mix",
  "description": "Optional description",
  "public": false
}
```

`name` is required. The response is the Spotify playlist object and uses status `201`.

### `GET /v1/playlists/{playlistID}/tracks`

Returns playlist track items. The public API keeps the `/tracks` route name, but the server calls Spotify's current `GET /v1/playlists/{playlist_id}/items` endpoint upstream. The server follows Spotify pagination and returns collected items.

Spotify may return `403` when the authenticated user is neither the owner nor a collaborator of the playlist.

```json
{
  "items": []
}
```

### `POST /v1/playlists/{playlistID}/tracks`

Adds tracks to a playlist. The public API keeps the `/tracks` route name, but the server calls Spotify's current `POST /v1/playlists/{playlist_id}/items` endpoint upstream. Spotify accepts at most 100 URIs per request, and the API rejects larger batches with `400`.

Request:

```json
{
  "uris": ["spotify:track:example"],
  "position": 0
}
```

`uris` is required and must contain 1 to 100 non-empty Spotify URIs. `position` is optional and is forwarded to Spotify. The response is the Spotify snapshot response and uses status `201`.

### `DELETE /v1/playlists/{playlistID}/tracks`

Removes explicit tracks from a playlist. Spotify accepts at most 100 URIs per request, and the API rejects larger batches with `400`.

Request:

```json
{
  "uris": ["spotify:track:example"],
  "snapshot_id": "optional-snapshot-id"
}
```

`uris` is required and must contain 1 to 100 non-empty Spotify URIs. The response is the Spotify snapshot response.

### `GET /v1/search/tracks?term=...`

Searches Spotify track candidates for instrumental versions of an original title. The server calls Spotify Search twice with fixed parameters `type=track`, `limit=10`, and `market=JP`:

- `q=<term> instrumental`
- `q=<term> カラオケ`

The latest candidate list is saved in process memory. Each returned item includes only track name, Spotify URL when Spotify provides one, artist names, and Spotify URI.

```json
{
  "items": [
    {
      "name": "Original Song - Instrumental",
      "url": "https://open.spotify.com/track/example",
      "artists": ["Example Artist"],
      "uri": "spotify:track:example"
    }
  ]
}
```

`term` is required.

### `DELETE /v1/search/tracks/cache`

Clears the latest track candidate search saved in process memory. The endpoint is idempotent and does not call Spotify.

Response:

```json
{
  "cleared": true
}
```

`cleared` is `true` when a cached search existed and was removed, otherwise `false`.

### `GET /v1/noLogin/search/playlists?keyword=...`

Searches public Spotify playlists without a user login. The server obtains an app-only Spotify token with Client Credentials, then calls Spotify Search with fixed query parameters `type=playlist`, `limit=10`, and `market=JP`. The response is plain text with one playlist per line and only number, name, and Spotify URL separated by tabs.

```text
1	Focus Playlist	https://open.spotify.com/playlist/example
```

`keyword` is required.

### `POST /v1/conversions`

Converts one source playlist into a new private instrumental playlist. `playlist_number` is the number returned by the latest `GET /v1/playlists` response for the same authenticated user.

Request:

```json
{
  "playlist_number": 1
}
```

If the server does not have a playlist list saved in process memory for the user, it fetches the user's Spotify playlists, saves them, and returns `409 text/plain` with only number, playlist title, and Spotify URL. No conversion is run for this response.

```text
1	Focus Playlist	https://open.spotify.com/playlist/example
```

For each source track, the server reuses the instrumental candidate search behavior and evaluates up to 20 candidates from `<title> instrumental` and `<title> カラオケ`. Source title comparisons use the text before the first `(` or `（` to avoid parenthetical subtitles blocking matches; candidate titles are compared in full. It first chooses a candidate whose title contains the source title, whose title contains `instrumental` or `インスト`, and whose artists include at least one source artist. If no such candidate exists, it falls back to the first candidate whose title contains the source title and contains `カラオケ` or `karaoke` in the candidate title.

Source tracks are read from Spotify playlist item objects returned by `GET /v1/playlists/{playlist_id}/items`. The converter uses the nested `item` track object from Spotify's current response shape, ignores large unused fields such as album metadata, and skips local or non-track items.

Successful response:

```json
{
  "created_playlist": {
    "title": "Focus Playlist Instrumental",
    "url": "https://open.spotify.com/playlist/created"
  },
  "added_count": 3,
  "not_found": [
    {
      "title": "Original Song",
      "url": "https://open.spotify.com/track/original"
    }
  ]
}
```

If no target tracks are found, `created_playlist` is `null`, `added_count` is `0`, and no empty playlist is created. `not_found` items contain only source track title and URL.

If the selected source playlist has no tracks, or has no playable Spotify track items, the endpoint returns `400 invalid_request` instead of a successful empty conversion.

## Error Responses

Errors use a stable JSON envelope.

```json
{
  "error": {
    "code": "spotify_api_error",
    "message": "spotify api request failed",
    "status": 403
  }
}
```

Common error codes:

- `missing_spotify_access_token`: missing or malformed `Authorization: Bearer ...` header.
- `spotify_access_token_expired`: stored Spotify access token is expired and the user needs to log in again.
- `invalid_json`: request body is not valid JSON.
- `invalid_request`: required query parameters or body fields are missing or invalid.
- `spotify_api_error`: Spotify returned a non-success status.
- `spotify_request_failed`: the upstream Spotify request failed before a Spotify error payload was available.
- `spotify_auth_error`: Spotify Accounts returned a non-success status.
- `spotify_auth_request_failed`: the Spotify Accounts request failed before a Spotify error payload was available.
- `spotify_client_credentials_missing`: Spotify Client ID or Client Secret is not configured.
- `spotify_auth_config_invalid`: Spotify Accounts client configuration is invalid.
- `spotify_redirect_uri_missing`: Spotify Redirect URI is not configured.
- `spotify_oauth_state_failed`: OAuth state could not be saved in process memory.
- `spotify_client_config_invalid`: `SPOTIFY_BASE_URL` is invalid.
- `token_store_failed`: token could not be saved in process memory.
- `token_not_found`: no in-memory token exists for the requested token id.
- `spotify_oauth_state_invalid`: OAuth callback state is missing, expired, or invalid.
- `spotify_oauth_code_missing`: OAuth callback code is missing.

Spotify access tokens and Spotify Client Secret values are never returned by error responses.

## Planned Conversion Endpoints

- `POST /v1/conversions/dry-run`
