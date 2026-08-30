# ADR-012 — Refresh-Token Rotation and Session Families

- Status: Accepted
- Date: 2026-08-29
- Owners: Project maintainer

## Context

Long-lived refresh credentials are replayable if stolen. Clients can also retry a refresh request after a network timeout, so the service must distinguish a safe retry from token reuse without creating two active descendants.

## Decision

Refresh tokens are 256-bit opaque random values stored only as SHA-256 hashes. Each login creates a session family. Every successful refresh atomically rotates the current session and creates a replacement. Any reuse of a rotated token revokes the entire family and emits an audit event.

## Alternatives considered

- Reusable refresh tokens: rejected because theft remains useful for the full TTL.
- Redis-only sessions: rejected because Redis is disposable and cannot be the source of truth.
- Silent acceptance of old tokens: rejected because it hides replay and creates competing session branches.

## Consequences

Refresh requires a PostgreSQL transaction and row lock or equivalent serializable guard. A client retry after a lost response may be treated as reuse; the client must restart authentication after family revocation.

## Failure modes and recovery

Expired, revoked, unknown, or reused tokens return generic `401`. Concurrent refresh permits at most one successful transition. Operators can revoke a family without knowing the raw token.

## Validation

Test normal rotation, concurrent requests, lost-response retry, expired token, revoked family, reuse detection, logout, logout-all, and password-reset revocation.

## Revisit when

The platform introduces device-bound credentials, sender-constrained tokens, or a formal session-management provider.
