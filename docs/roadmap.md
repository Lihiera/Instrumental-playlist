# Roadmap

## Phase 0: Repository Initialization

Status: Complete

- Initialize Git repository on `main`.
- Add project metadata, progress tracking, and agent definitions.
- Add base Go module.

## Phase 1: Web API and Configuration Foundation

Status: Complete

- Replace the CLI command foundation with an HTTP server foundation.
- Load runtime settings from `.env` and process environment variables.
- Remove dependency on OS user config/cache directories for application behavior.
- Add `/health` and `/v1/config` endpoints.
- Ensure secret values are never returned by public config responses.

## Phase 2: Spotify API Client Migration

Status: Complete

- Replace `internal/applemusic` with `internal/spotify`.
- Replace Apple Music Developer Token and Music User Token behavior with Spotify OAuth access token behavior.
- Read Spotify app settings from `.env`.
- Send upstream Spotify requests with `Authorization: Bearer <spotify_access_token>`.
- Handle Spotify JSON responses, `items`/`next` pagination, rate limits, retries, and Spotify error payloads.

## Phase 3: Spotify Playlist and Search REST APIs

- Add `GET /v1/playlists`.
- Add `POST /v1/playlists`.
- Add `GET /v1/playlists/{playlistID}/tracks`.
- Add `POST /v1/playlists/{playlistID}/tracks`.
- Add `DELETE /v1/playlists/{playlistID}/tracks`.
- Add `GET /v1/search/tracks?term=...`.
- Map upstream Spotify errors to stable JSON API responses.

## Phase 4: Instrumental Detection

- Implement default heuristic rules.
- Add scoring and exclusion reasons.
- Add tests for representative instrumental and non-instrumental metadata.

## Phase 5: Conversion REST APIs

- Add `POST /v1/conversions/dry-run`.
- Add `POST /v1/conversions`.
- Return adopted/excluded tracks and reasons as JSON.
- Create a new Spotify playlist and add accepted tracks for non-dry-run conversion.
- Split Spotify track additions into batches of at most 100 URIs.

## Phase 6: Hardening

- Add integration-style tests with mocked Spotify Web API.
- Improve partial failure handling.
- Document manual acceptance test steps and required Spotify scopes.
- Add API examples for local development.
