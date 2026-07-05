# Roadmap

## Phase 0: Repository Initialization

- Initialize Git repository on `main`.
- Add project metadata, progress tracking, and agent definitions.
- Add base Go module.

## Phase 1: Web API and Configuration Foundation

Status: Complete

- Replace the current CLI command foundation with an HTTP server foundation.
- Load runtime settings from `.env` and process environment variables.
- Remove dependency on OS user config/cache directories for application behavior.
- Add `/health` and `/v1/config` endpoints.
- Ensure secret values such as Developer Token are never returned by public config responses.

## Phase 2: Apple Music API Client

Status: Next

- Implement shared HTTP client with Developer Token authentication.
- Accept Music User Token from request headers for user-library operations.
- Handle JSON responses, pagination, rate limiting, retries, and Apple Music API errors.

## Phase 3: Playlist and Search REST APIs

- Add `GET /v1/playlists`.
- Add `POST /v1/playlists`.
- Add `DELETE /v1/playlists/{playlistID}`.
- Add `GET /v1/search/tracks?term=...`.
- Add playlist track read/add/remove endpoints.

## Phase 4: Instrumental Detection

- Implement default heuristic rules.
- Add scoring and exclusion reasons.
- Add tests for representative instrumental and non-instrumental metadata.

## Phase 5: Conversion REST APIs

- Add `POST /v1/conversions/dry-run`.
- Add `POST /v1/conversions`.
- Return adopted/excluded tracks and reasons as JSON.
- Create a new playlist and add accepted tracks for non-dry-run conversion.

## Phase 6: Hardening

- Add integration-style tests with mocked Apple Music API.
- Improve partial failure handling.
- Document manual acceptance test steps.
- Add API examples for local development.
