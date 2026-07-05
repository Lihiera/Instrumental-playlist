# AGENTS.md

## Project Direction

Instrumental Playlist is a Go Web API application for creating instrumental-only Apple Music playlists from existing playlists.

The current target is REST API behavior over HTTP, not CLI playlist operations. The executable in `cmd/instrumental-playlist` should start the HTTP server. Application wiring currently lives in `internal/app`.

## Current Implementation Baseline

- Runtime configuration is loaded from `.env` first, then process environment variables.
- Process environment variables override `.env` values.
- Supported settings are `HTTP_ADDR`, `APPLE_DEVELOPER_TOKEN`, and `APPLE_STOREFRONT`.
- Developer Token values must never be returned by API responses or logs.
- Music User Token should be accepted per request, using `X-Music-User-Token` for future Apple Music user-library endpoints.
- The app currently exposes `GET /health` and `GET /v1/config`.
- Gin is used for HTTP routing.
- Tests use `httptest` and run Gin in test mode.

## Agent Roles

Use the detailed role files under `.agents/` when a task needs role-specific focus:

- `.agents/architect.md`: API boundaries, configuration policy, token handling, and durable design decisions.
- `.agents/backend.md`: Go Web API implementation, Apple Music API client, handlers, and tests.
- `.agents/qa.md`: endpoint behavior, dry-run safety, secret redaction, and failure-mode coverage.

For ordinary implementation tasks, follow the combined guidance in this file and keep changes scoped.

## Coding Guidelines

- Prefer small packages with explicit boundaries under `internal/`.
- Keep HTTP handlers thin; move Apple Music behavior, conversion logic, and instrumental scoring into separate packages as they grow.
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
- Verify unsupported methods, malformed requests, missing tokens, secret redaction, pagination, retries, and partial Apple Music API failures as those features are added.
- Prefer mocked Apple Music API tests over tests that require a real Apple Music account.

## Documentation Guidelines

- Update `docs/progress.md` after meaningful implementation progress.
- Record durable architecture decisions in `docs/decisions.md`.
- Update `docs/api.md` when endpoints, request headers, response shapes, or error behavior change.
- Keep `README.md` user-facing and concise.

## Safety Rules

- Never commit `.env`, Developer Tokens, Music User Tokens, private keys, or generated secrets.
- Keep `.env.example` safe and placeholder-only.
- Do not depend on OS user config/cache/secrets directories for Web API runtime behavior.
- Do not overwrite or delete user work unless explicitly requested.
