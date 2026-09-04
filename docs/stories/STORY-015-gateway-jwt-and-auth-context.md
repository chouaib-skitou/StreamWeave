# STORY-015 — Verify JWTs and propagate trusted actor context

## User / system outcome

Protected requests are accepted only with a valid Identity-issued token and reach internal services with authenticated, non-spoofable context.

## Context

Identity owns signing keys and token issuance. Gateway performs coarse verification; resource owners perform final authorization.

## Acceptance criteria

- [ ] Given a valid EdDSA token, when its issuer, audience, time claims, `kid`, and required scope are valid, then the request may proceed.
- [ ] Given an expired, malformed, wrong-audience, wrong-algorithm, invalid-signature, or unknown-key token, when verification runs, then the request fails closed with 401.
- [ ] Given an unknown `kid`, when one bounded JWKS refresh cannot verify it, then no request is forwarded.
- [ ] Given client-supplied actor headers, when proxying occurs, then they are stripped and replaced only by trusted context.
- [ ] Given a downstream call, when authentication is attached, then it uses the Gateway service principal and never the human bearer.

## API changes

Bearer security and route scope classes are defined in the OpenAPI contract and route matrix.

## Event changes

None. Human credentials never enter events.

## Data changes

Disposable in-memory JWKS cache only.

## Failure cases

JWKS unavailable, unknown key, clock skew, invalid claims, missing scope, service-credential rotation failure.

## Observability

Auth-failure reason classes, JWKS refresh outcomes, and upstream authentication outcomes with no token values.

## Security considerations

EdDSA allowlist, issuer/audience pinning, context-header stripping, token redaction, short-lived machine credential.

## Test plan

JWT claim/signature matrix, key rotation/unknown `kid`, spoofed-header tests, service-context contract tests, and secret-redaction tests.

## Dependencies

Identity JWKS/token contract, Identity service-principal contract, and downstream service trust contract.

## Out of scope

Identity credential issuance, user administration, resource ownership, and private-key management.

## Validation evidence

Documentation gate: authentication design and ADR-016 reviewed against Identity ADR-011/013.
