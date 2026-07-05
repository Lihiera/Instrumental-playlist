# Progress

## Current Status

The project has completed Phase 2 of the Spotify migration. The application starts as a Go HTTP server, reads Spotify runtime settings from `.env` and process environment variables, and now has an internal Spotify Web API client foundation.

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

## Phase 2 Spotify Migration Completed

- Runtime config and `/v1/config` now expose Spotify settings with secret redaction.
- `internal/spotify` owns upstream bearer auth, response decoding, pagination, retries, and error parsing.
- Tests use `httptest` and do not require a real Spotify account.

## Next Actions

- Start Phase 3 playlist and search REST endpoints using the shared Spotify client.
- Extract `Authorization` access tokens in REST handlers and map missing-token errors to stable JSON responses.
- Add playlist write batching rules before implementing track-add endpoints.

## Open Questions

- Whether OAuth login and token refresh should be implemented server-side after the first Spotify API client migration.
- Whether CORS is needed for a future browser client.
- Whether conversion reports should be returned only in API responses or also saved to local files later.

## Verification Checklist

- `git status --short`: shows the Spotify migration changes.
- `git branch --show-current`: `main`
- `go env GOMOD`: `C:\Users\lgj46\Documents\Instrumental-playlist\go.mod`
- `go test ./...`: should pass.
- `go run ./cmd/instrumental-playlist`: starts the HTTP server.
