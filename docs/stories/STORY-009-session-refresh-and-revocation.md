# STORY-009 — Rotate Sessions and Revoke Credentials

## User / system outcome

As a client, I can refresh a session safely, log out, and recover from refresh-token replay.

## Context

Refresh tokens are opaque, hashed, 30-day credentials. Rotation must be atomic under retries and concurrency.

## Acceptance criteria

- [ ] Given an active refresh token, when refresh is called, then exactly one replacement session and token pair is created.
- [ ] Given concurrent refresh requests for one token, when both arrive, then at most one succeeds.
- [ ] Given a rotated refresh token, when it is reused, then the complete family is revoked and generic `401` is returned.
- [ ] Given logout or logout-all, when called repeatedly, then the operation is idempotent and durable.
- [ ] Given Redis is unavailable, when refresh is requested, then PostgreSQL session correctness is preserved and the configured rate-limit policy is observable.

## API changes

Implement `/v1/auth/refresh`, `/v1/auth/logout`, `/v1/auth/logout-all`, and `/v1/auth/sessions`.

## Event changes

Emit refresh, reuse-detected, logout, and revocation audit actions.

## Data changes

Implement session family, rotation links, revocation metadata, and unique refresh hash constraints.

## Failure cases

Expired, unknown, revoked, replayed, concurrently used tokens; PostgreSQL outage; Redis outage; lost HTTP response after commit.

## Observability

Measure refresh outcomes, reuse detection, family revocations, and session counts without token values.

## Security considerations

Use SHA-256 only for lookup of high-entropy opaque refresh tokens; never log or persist raw tokens.

## Test plan

Concurrency, integration, replay, expiry, logout, logout-all, lost-response, and race-detector tests.

## Out of scope

Device-bound credentials and WebAuthn.

## Dependencies

STORY-006, STORY-007, ADR-009, ADR-012, and the session data model.

## Validation evidence

Attach concurrent refresh test output and sanitized session lifecycle evidence.
