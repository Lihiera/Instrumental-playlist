# Progress

## Current Status

The project has completed Phase 4 of the Spotify migration and now has Spotify Authorization Code Flow support. The application starts as a Go HTTP server, reads Spotify runtime settings from `.env` and process environment variables, exposes Spotify playlist and track search REST endpoints, uses server-side Client Credentials for no-login public playlist search, and stores OAuth state plus user access/refresh tokens only in process memory for now.

## Completed

- Selected Go as the backend implementation language.
- Chose heuristic instrumental detection for v1.
- Chose new playlist creation as the default conversion target.
- Chose three initial agent roles: architect, backend, and qa.
- Confirmed the repository is on the `main` branch.
- Added the base Go module.
- Added an initial `cmd/` and `internal/` Go package skeleton.
- Implemented an initial CLI/config foundation with tests.
- Pivoted from CLI operations to REST API operations.
- Replaced the CLI command surface with HTTP server startup.
- Added `.env`/process-environment configuration for Spotify app settings.
- Added `.env.example`.
- Added `GET /health`.
- Added `GET /v1/config` with secret redaction by omission.
- Added Web API tests for startup wiring, config loading, health, config redaction, and unsupported methods.
- Added initial API documentation in `docs/api.md`.
- Replaced Apple Music configuration with `SPOTIFY_CLIENT_ID`, `SPOTIFY_CLIENT_SECRET`, `SPOTIFY_REDIRECT_URI`, and `SPOTIFY_BASE_URL`.
- Replaced Apple Music client wiring with `internal/spotify`.
- Added Spotify access-token request handling through `Authorization: Bearer <spotify_access_token>`.
- Added Spotify JSON response decoding, `items`/`next` pagination traversal, retry handling for rate limits and temporary server failures, and structured Spotify API error parsing.
- Added mocked Spotify Web API tests for auth headers, missing access tokens, secret redaction, pagination, retries, and non-retryable client errors.
- Added `GET /v1/playlists`.
- Added `POST /v1/playlists`.
- Added `GET /v1/playlists/{playlistID}/tracks`.
- Added `POST /v1/playlists/{playlistID}/tracks`.
- Added `DELETE /v1/playlists/{playlistID}/tracks`.
- Added `GET /v1/search/tracks?term=...`.
- Added stable JSON API error responses for missing bearer tokens, invalid JSON, invalid requests, Spotify API errors, and Spotify request failures.
- Added playlist write validation that limits add/remove requests to Spotify's 100 URI per request limit.
- Added mocked handler tests for bearer token handling, playlist pagination, playlist creation, search validation, track deletion request bodies, Spotify error mapping, access-token redaction, and add-track batch limits.
- Added `SPOTIFY_ACCOUNTS_BASE_URL` configuration.
- Added `POST /v1/auth/tokens` and `GET /v1/auth/tokens/{tokenID}` for process-memory token storage metadata.
- Added `GET /v1/auth/status` for checking whether a Spotify user token is currently stored in process memory.
- Added `POST /v1/auth/logout` for clearing process-memory Spotify user tokens.
- Added `GET /v1/noLogin/search/playlists?keyword=...` for no-login public playlist search with server-side app-only auth.
- Added tests for Spotify Client Credentials requests, missing Spotify app credentials, auth error redaction, and in-memory token storage behavior.
- Replaced unsafe `.env.example` credential values with placeholders.
- Added `GET /oauth/spotify/login` for Spotify Authorization Code Flow redirects with playlist scopes.
- Added `GET /oauth/spotify/callback` for OAuth state validation, authorization-code token exchange, and in-memory token metadata storage.
- Added process-memory OAuth state storage with one-time state consumption.
- Added mocked tests for login redirect contents, callback validation, token exchange form data, token metadata redaction, state replay rejection, missing callback code, and Spotify auth error redaction.

## Phase 2 Spotify Migration Completed

- Runtime config and `/v1/config` now expose Spotify settings with secret redaction.
- `internal/spotify` owns upstream bearer auth, response decoding, pagination, retries, and error parsing.
- Tests use `httptest` and do not require a real Spotify account.

## Phase 3 Spotify Playlist and Search APIs Completed

- Playlist and search handlers extract Spotify user access tokens from `Authorization: Bearer <spotify_access_token>`.
- Playlist and user search handlers use the latest in-memory OAuth user access token when `Authorization` is omitted.
- Playlist list and playlist track list endpoints follow Spotify pagination and return collected `items`.
- Playlist creation fetches the current Spotify user through `/v1/me` before creating the playlist under that user.
- Track add and delete endpoints require explicit Spotify URIs in the request body and reject batches over 100 URIs.
- Spotify upstream errors are mapped to stable JSON responses without returning access tokens.

## Spotify Auth and In-Memory Token Storage Added

- Public playlist search uses server-side Client Credentials without exposing token handling to clients.
- Access tokens and refresh tokens can be stored in process memory only.
- Token metadata endpoints do not return stored access tokens or refresh tokens.

## Phase 4 Spotify Authorization Code Flow Completed

- `GET /oauth/spotify/login` creates a process-memory OAuth state and redirects to Spotify Accounts `/authorize`.
- Login redirects include `playlist-read-private`, `playlist-modify-public`, and `playlist-modify-private` scopes.
- `GET /oauth/spotify/callback` consumes the stored state once, exchanges `code` at Spotify Accounts `/api/token`, saves access and refresh token values in process memory, and returns only token metadata.
- Callback errors and Spotify auth errors use stable JSON envelopes without exposing access tokens, refresh tokens, or the Spotify Client Secret.
- `GET /v1/auth/status` reports the current in-memory login status without exposing token values.
- `POST /v1/auth/logout` clears in-memory Spotify user tokens and is safe to call when already logged out.
- Playlist and user search endpoints now use the stored in-memory user access token automatically after login, while still allowing an explicit `Authorization: Bearer ...` header to override it.
- Playlist track listing now calls Spotify's current `GET /v1/playlists/{playlist_id}/items` upstream endpoint while keeping the app route as `GET /v1/playlists/{playlistID}/tracks`.

## Next Actions

- Start Phase 5 Redis token and state storage.
- Replace process-memory OAuth state and token storage with Redis-backed storage.
- Keep token values out of API responses and logs after the Redis migration.

## Open Questions

- Redis deployment details for local development and production-like environments.
- Whether CORS is needed for a future browser client.
- Whether conversion reports should be returned only in API responses or also saved to local files later.

## Verification Checklist

- `git status --short`: shows the Spotify migration changes.
- `git branch --show-current`: `main`
- `go env GOMOD`: `C:\Users\lgj46\Documents\Instrumental-playlist\go.mod`
- `go test ./...`: passes with workspace-local `APPDATA` and `GOCACHE`.
- `go run ./cmd/instrumental-playlist`: starts the HTTP server.
