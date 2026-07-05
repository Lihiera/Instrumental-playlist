# API Reference

## Runtime Settings

The server reads `.env` first, then process environment variables. Process environment variables take precedence.

Required or supported settings:

- `HTTP_ADDR`: HTTP listen address. Default: `:8080`.
- `SPOTIFY_CLIENT_ID`: Spotify application Client ID.
- `SPOTIFY_CLIENT_SECRET`: Spotify application Client Secret. This value is never returned by API responses.
- `SPOTIFY_REDIRECT_URI`: Redirect URI registered in the Spotify developer dashboard.
- `SPOTIFY_BASE_URL`: Spotify Web API base URL. Default: `https://api.spotify.com`.

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
  "spotify_base_url": "https://api.spotify.com"
}
```

## Spotify Request Headers

Future Spotify playlist and search endpoints will accept a Spotify user access token through the standard `Authorization` header.

```http
Authorization: Bearer replace-with-spotify-access-token
```

The access token must include the scopes required by the operation. Planned playlist operations need `playlist-read-private`, `playlist-modify-public`, and/or `playlist-modify-private`.

## Planned Spotify Endpoints

- `GET /v1/playlists`
- `POST /v1/playlists`
- `GET /v1/playlists/{playlistID}/tracks`
- `POST /v1/playlists/{playlistID}/tracks`
- `DELETE /v1/playlists/{playlistID}/tracks`
- `GET /v1/search/tracks?term=...`
- `POST /v1/conversions/dry-run`
- `POST /v1/conversions`
