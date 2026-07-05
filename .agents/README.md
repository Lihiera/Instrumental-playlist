# Agent Roles

This directory defines the initial working agents for the Instrumental Playlist project.

## Workflow

1. `architect` clarifies boundaries, interfaces, and design decisions.
2. `backend` implements Go code and REST API behavior.
3. `qa` validates tests, dry-run behavior, and failure modes.

Agents should update `docs/progress.md` when meaningful work is completed and record durable design decisions in `docs/decisions.md`.

## Shared Principles

- Prefer small, testable modules.
- Keep credentials and tokens out of the repository.
- Default to non-destructive playlist operations.
- Make instrumental inclusion/exclusion reasons visible to users.
- Use dry-run before making Apple Music library changes.
- Keep Developer Token in `.env` or process environment variables.
- Accept Music User Token per request instead of persisting it server-side.
