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

Status: Superseded by ADR-013

The Web API version will read the Apple Music Developer Token from `.env` or process environment variables. It will not generate Developer Tokens from `.p8` files in v1, and it will not depend on OS user config, cache, or secrets directories.

## ADR-009: Accept Music User Token Per Request

Status: Superseded by ADR-014

Apple Music user-library operations will receive the Music User Token through an HTTP request header such as `X-Music-User-Token`. The server will not persist user tokens in v1.

## ADR-010: Provide Public Config Without Secret Values

Status: Accepted

The Web API will expose public runtime configuration through `GET /v1/config`, but it will not return secret values. Secret visibility is represented only as configured/not-configured flags.

## ADR-011: Keep Apple Music API Access Behind an Internal Client

Status: Superseded by ADR-015

Apple Music API calls will go through `internal/applemusic`. The package owns Developer Token authentication, optional per-request Music User Token headers, JSON decoding, pagination traversal, retry behavior, and Apple Music API error parsing. REST handlers should depend on this client boundary instead of constructing raw Apple Music HTTP requests directly.

## ADR-012: Retry Only Rate Limits and Temporary Apple Music Failures by Default

Status: Superseded by ADR-015

The Apple Music client retries `429`, `500`, `502`, `503`, and `504` responses with a small exponential backoff and honors `Retry-After` when Apple provides it. Permanent `4xx` responses are returned without retry. Future write endpoints may override or narrow retry behavior if an operation is not safe to repeat.

## ADR-013: Migrate Playlist Editing From Apple Music to Spotify

Status: Accepted

Apple Music playlist editing will be replaced with Spotify playlist editing because the project cannot rely on Apple Music developer permissions. The product goal remains instrumental playlist conversion, but the upstream provider is now Spotify Web API.

## ADR-014: Accept Spotify Access Tokens Per Request

Status: Accepted

Spotify playlist and search endpoints will receive a user-authorized Spotify access token through `Authorization: Bearer <spotify_access_token>`. The server will not persist user access tokens in v1.

## ADR-015: Keep Spotify API Access Behind an Internal Client

Status: Accepted

Spotify Web API calls will go through `internal/spotify`. The package owns bearer token application, JSON decoding, Spotify `items`/`next` pagination traversal, retry behavior, and Spotify API error parsing. REST handlers should depend on this client boundary instead of constructing raw Spotify HTTP requests directly.

The Spotify client retries `429`, `500`, `502`, `503`, and `504` responses with a small exponential backoff and honors `Retry-After` when Spotify provides it. Permanent `4xx` responses are returned without retry.

## ADR-016: Keep Spotify Client Secret in `.env`

Status: Accepted

Spotify app credentials will be read from `.env` or process environment variables. `SPOTIFY_CLIENT_SECRET` must never be returned by API responses or logs. Public configuration may expose only whether the secret is configured.

## ADR-017: Support Client Credentials as App-Only Spotify Auth

Status: Accepted

The API will expose a Spotify Client Credentials token endpoint for app-only Spotify Web API calls that do not access user resources. This flow does not replace per-user OAuth access tokens for playlist read/write behavior. Playlist and conversion endpoints that operate on a user's library must continue to receive `Authorization: Bearer <spotify_access_token>`.

## ADR-018: Store Spotify Tokens in Process Memory for Now

Status: Accepted, planned to be superseded by ADR-020

Spotify access tokens and refresh tokens may be held in process memory for local development and early API wiring. The server will not persist these tokens to Redis, files, databases, OS secret stores, or logs in this phase. In-memory token storage is intentionally temporary and is lost when the process restarts.

## ADR-019: Implement Spotify Authorization Code Flow Endpoints

Status: Accepted

Phase 4 will add server-side Spotify Authorization Code Flow endpoints. `GET /oauth/spotify/login` will redirect users to Spotify authorization with playlist scopes, and `GET /oauth/spotify/callback` will validate OAuth state and exchange the authorization code for access and refresh tokens.

## ADR-020: Use Redis for OAuth State and Token Storage

Status: Accepted

Phase 5 will replace process-memory OAuth state and token storage with Redis-backed storage. Redis will hold OAuth state, token metadata, refresh tokens, and expiration data. Token values must remain absent from API responses and logs.
