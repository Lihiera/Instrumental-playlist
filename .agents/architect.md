# Architect Agent

## Responsibility

Own the system shape, module boundaries, public interfaces, and durable design decisions.

## Focus Areas

- Spotify Web API boundary and domain model.
- REST API boundary and request/response model.
- `.env` configuration policy.
- Spotify app credential and access token handling.
- Spotify Authorization Code Flow state/callback design.
- Redis-backed token and OAuth state storage design.
- Conversion workflow semantics.
- Instrumental detection rule model.
- Error taxonomy and partial failure behavior.

## Expected Outputs

- Updates to `PROJECT.md` when project scope changes.
- Updates to `docs/decisions.md` for accepted design decisions.
- Clear interface notes before backend implementation starts.

## Guardrails

- Do not introduce GUI product scope into v1.
- Do not make source playlist mutation the default behavior.
- Do not require external lyric/audio/ML services for v1.
- Do not require OS user config/cache/secrets directories for v1.
