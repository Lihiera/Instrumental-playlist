# AGENTS.md

## Project Direction

Instrumental Playlist is a Go Web API application for creating instrumental-only Spotify playlists from existing Spotify playlists.

The current target is REST API behavior over HTTP, not CLI playlist operations. The executable in `cmd/instrumental-playlist` should start the HTTP server. Application wiring currently lives in `internal/app`.

## Current Implementation Baseline

- Runtime configuration is loaded from `.env` first, then process environment variables.
- Process environment variables override `.env` values.
- Planned Spotify settings are `HTTP_ADDR`, `SPOTIFY_CLIENT_ID`, `SPOTIFY_CLIENT_SECRET`, `SPOTIFY_REDIRECT_URI`, and `SPOTIFY_BASE_URL`.
- Secret values must never be returned by API responses or logs.
- Spotify playlist and search endpoints should accept a user access token through `Authorization: Bearer <spotify_access_token>`.
- The app currently exposes `GET /health` and `GET /v1/config`.
- Gin is used for HTTP routing.
- Tests use `httptest` and run Gin in test mode.
- The current `internal/spotify` package owns Spotify Web API client behavior.

## Agent Roles

Use the detailed role files under `.agents/` when a task needs role-specific focus:

- `.agents/architect.md`: API boundaries, configuration policy, token handling, and durable design decisions.
- `.agents/backend.md`: Go Web API implementation, Spotify Web API client, handlers, and tests.
- `.agents/qa.md`: endpoint behavior, dry-run safety, secret redaction, and failure-mode coverage.

For ordinary implementation tasks, follow the combined guidance in this file and keep changes scoped.

## Coding Guidelines

- Prefer small packages with explicit boundaries under `internal/`.
- Keep HTTP handlers thin; move Spotify behavior, conversion logic, and instrumental scoring into separate packages as they grow.
- Keep configuration parsing deterministic and testable.
- Return public configuration through explicit DTOs such as `PublicConfig`; do not serialize secret-bearing structs directly.
- Treat dry-run behavior as a first-class workflow for conversion endpoints.
- Default to creating new playlists rather than mutating source playlists.
- Keep code comments concise and useful: explain non-obvious intent, edge cases, or external API constraints; avoid comments that merely restate the code.
- Do not introduce frontend UI, GUI, persistent user sessions, lyric APIs, audio analysis, or ML classifiers for v1.

## Testing Guidelines

- Run `go test ./...` after code changes.
- When local profile cache permissions interfere with Go commands on Windows, point `APPDATA` and `GOCACHE` at workspace-local temporary directories for that command.
- Use `httptest` for HTTP behavior.
- Verify unsupported methods, malformed requests, missing tokens, secret redaction, pagination, retries, Spotify rate limits, and partial Spotify API failures as those features are added.
- Prefer mocked Spotify Web API tests over tests that require a real Spotify account.

## Documentation Guidelines

- Update `docs/progress.md` after meaningful implementation progress.
- Record durable architecture decisions in `docs/decisions.md`.
- Update `docs/api.md` when endpoints, request headers, response shapes, or error behavior change.
- Keep `README.md` user-facing and concise.

## Safety Rules

- Never commit `.env`, Spotify Client Secrets, Spotify access tokens, refresh tokens, private keys, or generated secrets.
- Keep `.env.example` safe and placeholder-only.
- Do not depend on OS user config/cache/secrets directories for Web API runtime behavior.
- Do not overwrite or delete user work unless explicitly requested.
