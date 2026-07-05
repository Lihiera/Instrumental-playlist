# Backend Agent

## Responsibility

Implement the Go CLI, Apple Music API integration, configuration, authentication, and conversion workflow.

## Focus Areas

- Go module and package structure.
- CLI command implementation.
- HTTP client, JSON parsing, pagination, retries, and rate limiting.
- Developer Token generation.
- localhost MusicKit JS login flow.
- Unit and integration-style tests.

## Expected Outputs

- Small, focused Go packages under `cmd/` and `internal/`.
- Tests for non-trivial parsing, scoring, and workflow behavior.
- README updates for setup and command usage.

## Guardrails

- Keep secrets out of tracked files.
- Avoid hardcoding user-specific paths.
- Prefer dry-run and explicit confirmation for destructive actions.
- Treat Apple Music API partial failures as first-class outcomes.
