# EPIC-003 — Gateway Service

## Outcome

Provide a secure, observable, resilient public HTTP edge for StreamWeave without moving business ownership out of Identity or Orders.

## Scope

- Explicit public route registry and versioned OpenAPI facade.
- Request IDs, trace propagation, safe errors, and access logs.
- EdDSA/JWKS verification and coarse scopes.
- Authenticated service-context propagation.
- Redis route-aware rate limits and failure policy.
- Bounded upstream calls, health, metrics, container, Compose, Helm, and Kubernetes operations.

## Stories

- [STORY-014](./STORY-014-gateway-foundation.md) — Foundation, route registry, and public contract
- [STORY-015](./STORY-015-gateway-jwt-and-auth-context.md) — JWT verification and trusted context
- [STORY-016](./STORY-016-gateway-routing-and-errors.md) — Routing, timeouts, and Problem Details
- [STORY-017](./STORY-017-gateway-rate-limiting-and-resilience.md) — Rate limiting and resilience
- [STORY-018](./STORY-018-gateway-observability-and-operations.md) — Observability and production operations

## Dependencies

- Identity service contract and JWKS/session behavior.
- Orders public contract and ownership/idempotency behavior.
- Redis availability contract.
- Shared telemetry, container, Helm, CI, and release conventions.

## Definition of done

All stories pass unit, integration, contract, security, and end-to-end checks; the public contract and operational docs are synchronized; no secrets or direct database access exist; local and production deployment checks are reproducible.
