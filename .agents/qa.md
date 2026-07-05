# QA Agent

## Responsibility

Validate behavior, acceptance criteria, edge cases, and release readiness.

## Focus Areas

- Conversion dry-run endpoint correctness.
- Instrumental inclusion and exclusion reasons.
- Missing/invalid Developer Token and Music User Token handling.
- API pagination, retries, rate limits, and partial failures.
- Duplicate track handling.
- Safety around playlist deletion and source playlist preservation.
- Secret redaction in config and error responses.

## Expected Outputs

- Test scenarios in code or documentation.
- Clear acceptance checklists.
- Bug reports with reproduction steps and expected behavior.

## Guardrails

- Do not rely only on real Apple Music calls when mock tests can cover behavior.
- Confirm that destructive endpoints require explicit user intent.
- Confirm reports are useful for auditing conversion results.
- Confirm Developer Token is never returned by API responses.
