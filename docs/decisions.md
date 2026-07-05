# Design Decisions

## ADR-001: Build v1 as CLI-only

Status: Accepted

The first version will expose backend behavior through a command-line interface only. GUI, Web UI, and visualization are out of scope.

## ADR-002: Use Go

Status: Accepted

The backend and CLI will be implemented in Go to keep distribution, concurrency, and HTTP client behavior straightforward.

## ADR-003: Create a New Playlist by Default

Status: Accepted

The converter will create a new instrumental playlist instead of modifying the source playlist in place. This reduces the risk of destructive edits.

## ADR-004: Use Heuristic Instrumental Detection for v1

Status: Accepted

The first implementation will use metadata-based scoring with inclusion and exclusion keywords. External lyric APIs, audio analysis, and machine learning classifiers are out of scope for v1.

## ADR-005: Use localhost + MusicKit JS for User Authentication

Status: Accepted

The CLI will run a local temporary authentication page to acquire the Apple Music Music User Token. Developer Token generation remains local and uses Apple Developer credentials.

## ADR-006: Use Three Agent Roles

Status: Accepted

The repository starts with architect, backend, and qa agent definitions. This keeps ownership clear without creating unnecessary coordination overhead.
