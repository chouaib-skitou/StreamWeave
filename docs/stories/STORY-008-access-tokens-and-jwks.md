# STORY-008 — Issue Access Tokens and Publish JWKS

## User / system outcome

As a gateway or service, I can verify Identity-issued access tokens using public keys without receiving the private signing key.

## Context

Identity signs human and machine JWTs with Ed25519/EdDSA and publishes active public keys with stable key IDs.

## Acceptance criteria

- [ ] Given valid signing configuration, when a token is issued, then it contains the required claims and a 10-minute default TTL.
- [ ] Given a valid key ID, when JWKS is requested, then only public Ed25519 keys are returned.
- [ ] Given a malformed, expired, wrongly issued, wrongly addressed, or wrongly algorithmed token, when a verifier checks it, then it rejects the token.
- [ ] Given a key rotation overlap, when an old token is checked, then its public key remains discoverable until policy expiry.
- [ ] Given an unknown key ID, when verification retries JWKS once and still cannot find the key, then verification fails closed.

## API changes

Implement `GET /.well-known/jwks.json` and token response claims in the login/refresh contracts.

## Event changes

Audit successful and failed token issuance where applicable.

## Data changes

No private key material is stored in PostgreSQL; key-provider state is external to the database.

## Failure cases

Missing key, key-provider outage, unknown `kid`, JWKS serialization error, and algorithm confusion attempt.

## Observability

Metrics cover issuance, verification outcome, JWKS refresh, unknown key IDs, and key-provider failures.

## Security considerations

Algorithm allowlist, issuer/audience validation, no private-key logs, and no private key in container layers or ConfigMaps.

## Test plan

Cryptographic, claim-validation, rotation, JWKS schema, cache-refresh, and secret-scanning tests.

## Out of scope

Gateway middleware implementation is owned by the Gateway service.

## Dependencies

STORY-006, ADR-009, ADR-011, and the Identity OpenAPI contract.

## Validation evidence

Attach public JWKS output and token-claim tests with secrets redacted.
