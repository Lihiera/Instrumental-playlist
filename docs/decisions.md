# Design Decisions

## ADR-001: Build v1 as CLI-only

Status: Superseded by ADR-007

The first version will expose backend behavior through a command-line interface only. GUI, Web UI, and visualization are out of scope.

## ADR-002: Use Go

Status: Accepted

The backend and HTTP server will be implemented in Go to keep distribution, concurrency, and HTTP client behavior straightforward.

## ADR-003: Create a New Playlist by Default

Status: Accepted

The converter will create a new instrumental playlist instead of modifying the source playlist in place. This reduces the risk of destructive edits.

## ADR-004: Use Heuristic Instrumental Detection for v1

Status: Accepted

The first implementation will use metadata-based scoring with inclusion and exclusion keywords. External lyric APIs, audio analysis, and machine learning classifiers are out of scope for v1.

## ADR-005: Use localhost + MusicKit JS for User Authentication

Status: Superseded by ADR-009

The CLI will run a local temporary authentication page to acquire the Apple Music Music User Token. Developer Token generation remains local and uses Apple Developer credentials.

## ADR-006: Use Three Agent Roles

Status: Accepted

The repository starts with architect, backend, and qa agent definitions. This keeps ownership clear without creating unnecessary coordination overhead.

## ADR-007: Expose v1 Behavior Through REST APIs

Status: Accepted

The project will be restructured as a Go Web API application. Playlist operations, track search, and conversion workflows will be executed through REST endpoints instead of CLI subcommands.

## ADR-008: Load Developer Token From `.env`

Status: Accepted

The Web API version will read the Apple Music Developer Token from `.env` or process environment variables. It will not generate Developer Tokens from `.p8` files in v1, and it will not depend on OS user config, cache, or secrets directories.

## ADR-009: Accept Music User Token Per Request

Status: Accepted

Apple Music user-library operations will receive the Music User Token through an HTTP request header such as `X-Music-User-Token`. The server will not persist user tokens in v1.

## ADR-010: Provide Public Config Without Secret Values

Status: Accepted

The Web API will expose public runtime configuration through `GET /v1/config`, but it will not return secret values. Developer Token visibility is represented only as a boolean configured/not-configured flag.
