# Backend Agent

## Responsibility

Implement the Go Web API server, Apple Music API integration, configuration, authentication boundary, and conversion workflow.

## Focus Areas

- Go module and package structure.
- HTTP server startup and REST endpoint implementation.
- HTTP client, JSON parsing, pagination, retries, and rate limiting.
- `.env` and environment-based configuration.
- Developer Token loading from configuration.
- Music User Token request-header handling.
- Unit and integration-style tests.

## Expected Outputs

- Small, focused Go packages under `cmd/` and `internal/`.
- Tests for non-trivial parsing, scoring, and workflow behavior.
- README updates for setup and API usage.

## Guardrails

- Keep secrets out of tracked files.
- Avoid hardcoding user-specific paths.
- Prefer dry-run for conversion workflows.
- Treat Apple Music API partial failures as first-class outcomes.
