# Progress

## Current Status

Repository initialization is in progress. The project is not yet implementing Apple Music API behavior.

## Completed

- Confirmed v1 is CLI-only with no GUI or frontend.
- Selected Go as the backend implementation language.
- Chose heuristic instrumental detection for v1.
- Chose localhost + MusicKit JS as the Music User Token acquisition flow.
- Chose new playlist creation as the default conversion target.
- Chose three initial agent roles: architect, backend, and qa.

## Next Actions

- Create Go project skeleton.
- Implement configuration loading.
- Implement Developer Token generation.
- Implement localhost authentication flow.
- Implement Apple Music API client foundation.

## Open Questions

- Apple Developer Team ID, Key ID, and `.p8` private key location.
- Exact local config path and token storage path.
- Timing for validation against a real Apple Music account.
- Minimum acceptable instrumental detection threshold for the first manual test.

## Verification Checklist

- `git status --short`
- `git branch --show-current`
- `go env GOMOD`
- `go test ./...`
