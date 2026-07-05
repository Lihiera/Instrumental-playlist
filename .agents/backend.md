# Backend Agent

## Responsibility

Implement the Go Web API server, Spotify Web API integration, configuration, authentication boundary, and conversion workflow.

## Focus Areas

- Go module and package structure.
- HTTP server startup and REST endpoint implementation.
- Gin router and middleware usage for HTTP API behavior.
- HTTP client, JSON parsing, pagination, retries, and rate limiting.
- `.env` and environment-based configuration.
- Spotify app credential loading from configuration.
- Spotify access token request-header handling.
- Spotify Authorization Code Flow login/callback endpoints.
- Redis-backed OAuth state and token storage.
- Concise code comments for non-obvious intent, edge cases, and Spotify API constraints.
- Unit and integration-style tests.

## Expected Outputs

- Small, focused Go packages under `cmd/` and `internal/`.
- Tests for non-trivial parsing, scoring, and workflow behavior.
- README updates for setup and API usage.

## Guardrails

- Keep secrets out of tracked files.
- Avoid hardcoding user-specific paths.
- Use Gin consistently for route registration, middleware, JSON responses, and method handling.
- Add comments only where they clarify intent or constraints; avoid comments that merely restate the code.
- Keep OAuth state validation and Redis storage boundaries explicit and testable.
- Prefer dry-run for conversion workflows.
- Treat Spotify Web API partial failures as first-class outcomes.
