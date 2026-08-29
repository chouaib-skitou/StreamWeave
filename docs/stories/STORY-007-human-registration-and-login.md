# STORY-007 — Implement Demo Registration and Human Login

## User / system outcome

As a customer, I can authenticate with credentials and receive a short-lived access token plus an opaque refresh token.

## Context

The blueprint requires Argon2id, generic credential failures, default customer role assignment, durable sessions, and safe audit publication.

## Acceptance criteria

- [ ] Given demo mode is disabled, when registration is requested, then the endpoint is unavailable without creating a user.
- [ ] Given valid demo registration, when the request is submitted, then a normalized unique user and Argon2id password hash are persisted.
- [ ] Given an unknown email or wrong password, when login is attempted, then the same generic `401` response is returned.
- [ ] Given valid credentials, when login succeeds, then a durable session family is created and tokens are returned without secret leakage.
- [ ] Given excessive login failures, when the configured rate limit is exceeded, then `429` is returned and the event is observable.

## API changes

Implement `POST /.well-known/register` and `POST /v1/auth/login`.

## Event changes

Emit `register.succeeded`, `login.succeeded`, or sanitized `login.failed` audit actions.

## Data changes

Write `users`, role assignment, `sessions`, `security_audit`, and `outbox_events` in the appropriate local transactions.

## Failure cases

Unknown account, disabled account, malformed credentials, database outage, rate-limit outage, and audit publication outage.

## Observability

Record outcome classes, latency, rate limiting, and correlation IDs; never email, password, token, or raw IP.

## Security considerations

Benchmark Argon2id without weakening it to satisfy latency. Use constant-shape credential handling and generic errors.

## Test plan

Domain, handler, PostgreSQL integration, rate-limit, timing-shape, security-log, and contract tests.

## Out of scope

Refresh rotation, role administration, and password reset are separate stories.

## Dependencies

STORY-006, the Identity PRD, the data model, and the authentication ADRs.

## Validation evidence

Attach passing tests and a sanitized request/response demonstration.
