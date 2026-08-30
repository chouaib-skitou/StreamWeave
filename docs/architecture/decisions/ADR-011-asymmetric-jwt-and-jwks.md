# ADR-011 — Asymmetric JWT Signing and JWKS

- Status: Accepted
- Date: 2026-08-29
- Owners: Project maintainer

## Context

Multiple services must verify access tokens, but only Identity should be able to mint them. Key rotation must not invalidate every token immediately or distribute private signing material.

## Decision

Identity signs access tokens with Ed25519/EdDSA and publishes public keys through `GET /.well-known/jwks.json`. Tokens contain an explicit `kid`, issuer, audience, expiry, issued-at, subject, JTI, roles, and scopes. Verifiers use an algorithm allowlist and refresh JWKS on an unknown key ID. Private keys are provided through a secret-file or KMS-backed port and never through Git, a ConfigMap, or JWKS.

## Alternatives considered

- HMAC: rejected because verifiers would possess the signing secret.
- RSA: viable but not selected because Ed25519 provides a smaller modern key/signature footprint for this service.
- Static key without JWKS: rejected because rotation would require coordinated redeployments.

## Consequences

Verifiers need safe JWKS caching and unknown-key refresh behavior. Old public keys remain available until tokens signed with them can no longer be accepted. Key configuration errors fail readiness and stop new issuance.

## Failure modes and recovery

Unknown or unverifiable key IDs return unauthorized after one JWKS refresh. Key-provider outage prevents new token issuance and is visible through readiness and metrics. Emergency JTI revocation is checked separately from signature validation.

## Validation

Test key rotation, old-key verification during overlap, unknown `kid`, algorithm confusion attempts, malformed claims, JWKS containing public material only, and private-key absence from logs/responses.

## Revisit when

The deployment adopts a managed KMS/HSM, external federation, or a different cryptographic compliance requirement.
