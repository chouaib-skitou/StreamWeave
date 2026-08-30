# ADR-013 — Service-to-Service Identity

- Status: Accepted
- Date: 2026-08-29
- Owners: Project maintainer

## Context

Internal service calls need authentication without propagating human bearer tokens through Kafka or reusing customer credentials. The blueprint requires service principals, a separate audience, short TTL, and machine scopes but leaves issuance mechanics open.

## Decision

Identity provisions service principals out of band. An internal-only client-credentials endpoint verifies a stored credential hash and issues a short-lived EdDSA JWT for the principal's dedicated machine audience and allowed scopes. The public gateway never exposes this endpoint. Service credentials are rotated outside the application and are never returned or logged. One replacement credential may overlap the previous credential for a bounded rotation window; after that window the previous credential is retired.

## Alternatives considered

- Reuse human access tokens: rejected because lifecycle, audience, and audit semantics differ.
- Mutual TLS only: deferred as a deployment-level hardening layer, not the sole application identity in the portfolio baseline.
- Static API keys: rejected because they lack audience, expiry, and scope claims.

## Consequences

Each service needs a safe credential reference and token-verification configuration. The service-token route is a sensitive internal boundary and must be covered by network policy, rate limiting, audit, and security tests. Requested scopes must be a subset of the principal's grants; Identity returns `403` rather than silently reducing or expanding the request.

## Failure modes and recovery

Unknown, disabled, or insufficiently scoped principals receive generic unauthorized/forbidden outcomes. Credential rotation disables the old principal without changing human sessions. PostgreSQL or key-provider failure prevents new machine tokens.

## Validation

Test separate audiences, scope-subset enforcement, disabled principals, credential rotation, public-route exclusion, expiry, audit emission, and attempted human-token reuse.

## Revisit when

The platform adopts workload identity, mTLS as the mandatory trust root, SPIFFE/SPIRE, or a managed authorization service.
