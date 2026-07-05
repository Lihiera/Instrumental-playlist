# Progress

## Current Status

The project has completed the Phase 1 migration from a CLI-first backend to a Web API application foundation. The application now starts as a Go HTTP server and reads runtime settings from `.env` and process environment variables.

Apple Music API behavior is still not implemented. The next implementation task is Phase 2: add an Apple Music API client with Developer Token authentication and per-request Music User Token handling.

## Completed

- Selected Go as the backend implementation language.
- Chose heuristic instrumental detection for v1.
- Chose new playlist creation as the default conversion target.
- Chose three initial agent roles: architect, backend, and qa.
- Confirmed the repository is on the `main` branch.
- Added the base Go module.
- Added an initial `cmd/` and `internal/` Go package skeleton.
- Implemented an initial CLI/config foundation with tests.
- Decided to pivot from CLI operations to REST API operations.
- Decided that Developer Token will be read from `.env`.
- Decided that OS user config/cache/secrets directories are not required for the Web API version.
- Replaced the CLI command surface with HTTP server startup.
- Added `.env`/process-environment configuration for `HTTP_ADDR`, `APPLE_DEVELOPER_TOKEN`, `APPLE_STOREFRONT`, and `INSTRUMENTAL_THRESHOLD`.
- Added `.env.example`.
- Added `GET /health`.
- Added `GET /v1/config` with Developer Token redaction by omission.
- Added Web API tests for startup wiring, config loading, health, config redaction, and unsupported methods.
- Added initial API documentation in `docs/api.md`.

## Phase 1 Migration Completed

- `cmd/instrumental-playlist` starts the HTTP server.
- `internal/app` now wires runtime configuration into HTTP handlers.
- OS user config/cache/secrets directory behavior has been removed.
- CLI `config paths` and `config show` behavior has been removed.
- README and API documentation describe the Web API skeleton.

## Next Actions

- Define Apple Music API client interfaces for later endpoint implementation.
- Implement shared Apple Music HTTP client with Developer Token authentication.
- Define request handling for `X-Music-User-Token`.
- Add mocked tests for Apple Music API error and pagination behavior.

## Open Questions

- Exact client-side flow for obtaining Music User Token before calling this API.
- Whether CORS is needed for a future browser client.
- Minimum acceptable instrumental detection threshold for the first manual test.
- Whether conversion reports should be returned only in API responses or also saved to local files later.

## Verification Checklist

- `git status --short`: shows the Web API migration changes.
- `git branch --show-current`: `main`
- `go env GOMOD`: `C:\Users\lgj46\Documents\Instrumental-playlist\go.mod`
- `go test ./...`: should pass.
- `go run ./cmd/instrumental-playlist`: starts the HTTP server.
