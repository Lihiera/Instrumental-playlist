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
- `REDIS_ADDR`: Redis address for Phase 5 token and OAuth state storage.
- `REDIS_PASSWORD`: Optional Redis password.
- `REDIS_DB`: Optional Redis database number.

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

## Planned Authorization Code Flow Endpoints

### `GET /oauth/spotify/login`

Phase 4 will add a login endpoint that creates an OAuth `state` value and redirects the user to Spotify Accounts authorization. The authorization request must include the playlist scopes needed by this API.

### `GET /oauth/spotify/callback`

Phase 4 will add a callback endpoint that validates `state`, exchanges Spotify's authorization `code` for access and refresh tokens, and stores token metadata without returning token values.

Phase 5 will move callback state and token storage from process memory to Redis.

## Spotify Request Headers

Spotify playlist and search endpoints accept a Spotify user access token through the standard `Authorization` header.

```http
Authorization: Bearer replace-with-spotify-access-token
```

The access token must include the scopes required by the operation. Playlist operations need `playlist-read-private`, `playlist-modify-public`, and/or `playlist-modify-private`.

## Spotify Endpoints

### `GET /v1/playlists`

Returns the current Spotify user's playlists. The server follows Spotify pagination and returns collected items.

```json
{
  "items": []
}
```

### `POST /v1/playlists`

Creates a playlist for the current Spotify user. The server fetches `/v1/me` from Spotify to find the user id before creating the playlist.

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

Returns playlist track items. The server follows Spotify pagination and returns collected items.

```json
{
  "items": []
}
```

### `POST /v1/playlists/{playlistID}/tracks`

Adds tracks to a playlist. Spotify accepts at most 100 URIs per request, and the API rejects larger batches with `400`.

Request:

```json
{
  "uris": ["spotify:track:example"],
  "position": 0
}
```

`uris` is required and must contain 1 to 100 non-empty Spotify URIs. `position` is optional. The response is the Spotify snapshot response and uses status `201`.

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

Searches Spotify tracks.

```json
{
  "items": []
}
```

`term` is required.

### `GET /v1/noLogin/search/playlists?keyword=...`

Searches public Spotify playlists without a user login. The server obtains an app-only Spotify token with Client Credentials, then calls Spotify Search with fixed query parameters `type=playlist`, `limit=10`, and `market=JP`.

```json
{
  "items": []
}
```

`keyword` is required.

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
- `invalid_json`: request body is not valid JSON.
- `invalid_request`: required query parameters or body fields are missing or invalid.
- `spotify_api_error`: Spotify returned a non-success status.
- `spotify_request_failed`: the upstream Spotify request failed before a Spotify error payload was available.
- `spotify_auth_error`: Spotify Accounts returned a non-success status.
- `spotify_auth_request_failed`: the Spotify Accounts request failed before a Spotify error payload was available.
- `spotify_client_credentials_missing`: Spotify Client ID or Client Secret is not configured.
- `spotify_auth_config_invalid`: Spotify Accounts client configuration is invalid.
- `spotify_client_config_invalid`: `SPOTIFY_BASE_URL` is invalid.
- `token_store_failed`: token could not be saved in process memory.
- `token_not_found`: no in-memory token exists for the requested token id.
- `spotify_oauth_state_invalid`: OAuth callback state is missing, expired, or invalid.
- `spotify_oauth_code_missing`: OAuth callback code is missing.
- `redis_unavailable`: Redis-backed token or OAuth state storage is unavailable.

Spotify access tokens and Spotify Client Secret values are never returned by error responses.

## Planned Conversion Endpoints

- `POST /v1/conversions/dry-run`
- `POST /v1/conversions`
