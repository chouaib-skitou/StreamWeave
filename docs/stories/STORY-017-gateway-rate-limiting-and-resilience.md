# STORY-017 — Enforce route-aware rate limits safely

## User / system outcome

Abusive or excessive traffic is bounded without converting Redis into business state or silently disabling protection during failure.

## Context

Anonymous auth routes and protected order mutations have different abuse profiles. Redis is disposable.

## Acceptance criteria

- [ ] Given traffic within policy, when Redis evaluates the route limit, then the request proceeds.
- [ ] Given an exhausted limit, when a request arrives, then Gateway returns RFC 9457 `429` with `Retry-After`.
- [ ] Given a limiter failure on an auth or mutation route, when policy cannot be enforced, then Gateway fails closed with safe `503`.
- [ ] Given a health request, when Redis is unavailable, then liveness remains local and does not create a restart loop.
- [ ] Given limits, when telemetry is emitted, then labels remain route/limit class only.

## API changes

429/503 behavior and rate-limit classes are in the route matrix and OpenAPI responses.

## Event changes

None.

## Data changes

TTL-bound namespaced Redis counters only.

## Failure cases

Redis unavailable, timeout, authentication failure, saturated connection pool, clock skew, malformed limit key input.

## Observability

Rejections and limiter errors by route and limit class; no user/IP/token labels.

## Security considerations

No unlimited fallback, bounded key input, Redis TLS/auth in staging/production, and no sensitive key material in logs.

## Test plan

Atomic counter tests, expiry tests, concurrency tests, failure-mode tests, `Retry-After` contract tests, and abuse scenario tests.

## Dependencies

Redis operational contract, environment secret/TLS policy, route matrix, and monitoring metric vocabulary.

## Out of scope

Distributed business locks, order idempotency, and adaptive traffic management.

## Validation evidence

Documentation gate: ADR-017 and failure runbook reviewed for fail-closed consistency.
