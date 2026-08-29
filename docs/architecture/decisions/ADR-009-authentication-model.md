# ADR-009 — Authentication Model

- Status: Accepted
- Date: 2026-08-29
- Owners: Project maintainer

## Context

The platform needs human authentication without coupling the gateway or business services to the Identity database. Authentication must support short-lived access, durable sessions, logout, refresh, and security audit under duplicate delivery and dependency failure.

## Decision

Identity owns users, credentials, sessions, and token issuance. Human access tokens are short-lived EdDSA JWTs. Refresh credentials are opaque, durable-session-backed tokens. The gateway verifies access tokens and coarse scopes; each business service applies resource ownership and domain authorization.

## Alternatives considered

- External identity provider: deferred because the portfolio needs an independently demonstrable Identity boundary.
- Symmetric JWTs: rejected because every verifier would need a shared signing secret.
- Server-side opaque access tokens: rejected for v1 because it adds a synchronous lookup to every request.

## Consequences

- Business services can verify access tokens locally through JWKS.
- Revocation is not instant for every access token; emergency JTI revocation is provided through Redis.
- Refresh-session correctness remains dependent on Identity PostgreSQL availability.

## Failure modes and recovery

Expired, forged, wrongly scoped, wrongly addressed, or revoked tokens are rejected. PostgreSQL outage prevents durable session issuance. Redis outage fails closed for emergency JTI checks but does not erase durable sessions.

## Validation

Security tests cover wrong credentials, forged signatures, invalid issuer/audience, expiry, JTI revocation, session revocation, and ownership enforcement in the consuming service.

## Revisit when

The platform requires federation, multi-region sessions, regulated identity proofing, or a hosted identity provider.
