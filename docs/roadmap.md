# Roadmap

## Phase 0: Repository Initialization

- Initialize Git repository on `main`.
- Add project metadata, progress tracking, and agent definitions.
- Add base Go module.

## Phase 1: CLI and Configuration Foundation

- Define CLI command structure.
- Implement configuration loading from file and environment variables.
- Define local paths for config, tokens, reports, and secrets.

## Phase 2: Authentication

- Generate Apple Music Developer Token from Team ID, Key ID, and `.p8` private key.
- Implement `auth login` with localhost + MusicKit JS.
- Store Music User Token locally outside the repository.

## Phase 3: Apple Music API Client

- Implement shared HTTP client with authentication headers.
- Add playlist list/create/delete operations.
- Add catalog search.
- Add playlist track read/add/remove operations.
- Handle pagination, rate limiting, retries, and API errors.

## Phase 4: Instrumental Detection

- Implement default heuristic rules.
- Add scoring, exclusion reasons, and configurable threshold.
- Add tests for representative instrumental and non-instrumental metadata.

## Phase 5: Conversion Workflow

- Implement `convert --source <playlist-id> --name <new-name>`.
- Add `--dry-run` output with adopted/excluded tracks and reasons.
- Create new playlist and add accepted tracks.
- Save conversion reports for auditing and reruns.

## Phase 6: Hardening

- Add integration-style tests with mocked Apple Music API.
- Improve partial failure handling.
- Document manual acceptance test steps.
