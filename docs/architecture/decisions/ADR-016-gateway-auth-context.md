# ADR-016: Verify human tokens at Gateway and propagate safe service context

- **Status:** accepted
- **Date:** 2026-09-04
- **Owners:** Platform architecture and Identity
- **Scope:** Gateway-to-service authentication

## Context

Identity signs human access tokens with EdDSA and publishes JWKS. Orders needs an authenticated actor but must remain the authority for resource ownership. Forwarding an end-user bearer token increases replay and disclosure risk; trusting arbitrary `X-*` headers enables spoofing.

## Decision

Gateway validates the incoming human token against the configured Identity issuer, audience, algorithm allowlist, claims, and key ID. Gateway strips all incoming internal identity headers and calls downstream services with a dedicated short-lived `svc_gateway` service credential. It forwards only a documented safe actor context and correlation/trace headers. Downstream services authenticate the service credential and may trust the context for request attribution, but they still perform final authorization.

Human tokens must never be forwarded through Kafka, placed in logs/traces, or copied into downstream error responses.

## Consequences

- Identity remains the private-key owner.
- Orders can enforce ownership without a shared identity database.
- Gateway must handle JWKS freshness and service-credential rotation.
- Service-to-service trust and context headers require contract tests.

## Alternatives rejected

- **Symmetric JWT verification:** would distribute a signing secret and enlarge blast radius.
- **Forward the human bearer token:** increases replay and token leakage risk.
- **Trust client headers:** permits impersonation.
- **Gateway-only authorization:** cannot correctly enforce order ownership or business policy.

## Failure modes considered

- JWKS unavailable or an unknown `kid` results in fail-closed authentication.
- A spoofed actor header is removed before an internal call.
- Gateway service-credential failure blocks downstream access rather than trusting unsigned context.

## Validation

The decision is reflected by `gateway-authentication.md`, `gateway.openapi.yaml`, and STORY-015.
