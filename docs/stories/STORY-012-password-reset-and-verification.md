# STORY-012 — Implement Password Reset and Email Verification

## User / system outcome

As a user, I can complete a simulated password reset or email verification without the platform exposing account existence or token values.

## Context

The blueprint requires hashed one-time tokens and simulated mail delivery. Reset must revoke active sessions.

## Acceptance criteria

- [ ] Given any email, when a reset is requested, then the same `202` response is returned without account enumeration.
- [ ] Given a valid reset token, when a new password is submitted, then the password hash changes and active session families are revoked.
- [ ] Given an expired, consumed, or unknown token, when confirmation is attempted, then generic unauthorized is returned and no state changes.
- [ ] Given a valid verification token, when confirmation is submitted, then only the owning user's verification state changes.
- [ ] Given mailer failure, when a token is created, then delivery retries without exposing the token through logs or public API responses.

## API changes

Implement password-reset request/confirm and email-verification confirm endpoints.

## Event changes

Emit reset-requested, reset-completed, and email-verification-completed audit actions.

## Data changes

Implement one-time hashed token tables and expiry/consumption indexes.

## Failure cases

Enumeration attempts, token replay, expiry, mailer outage, database outage, password policy failure, and session-revocation failure.

## Observability

Record outcome classes and delivery backlog without email addresses beyond approved hashed identifiers.

## Security considerations

Use strong password policy, Argon2id, generic responses, single-use tokens, and no token material in telemetry.

## Test plan

API, token lifecycle, expiry, replay, session revocation, mailer retry, and secret-scanning tests.

## Out of scope

Real email delivery, external identity federation, and passwordless authentication.

## Dependencies

STORY-006, STORY-007, STORY-009, the token data model, and the simulated mailer port.

## Validation evidence

Attach token lifecycle and enumeration-resistance tests.
