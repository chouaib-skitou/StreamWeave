# Architecture Decision Records

Store decision records here using the format `ADR-NNN-short-title.md`.

An ADR is required when a choice affects service boundaries, ownership, contracts, consistency, reliability, security, deployment, or a reusable architectural pattern. Each ADR records context, decision, alternatives, consequences, failure modes, and validation.

Identity decisions:

- [ADR-009 — Authentication model](ADR-009-authentication-model.md)
- [ADR-010 — RBAC and explicit scopes](ADR-010-rbac-and-scopes.md)
- [ADR-011 — Asymmetric JWT signing and JWKS](ADR-011-asymmetric-jwt-and-jwks.md)
- [ADR-012 — Refresh-token rotation and session families](ADR-012-refresh-token-rotation.md)
- [ADR-013 — Service-to-service identity](ADR-013-service-to-service-identity.md)
- [ADR-014 — Security audit delivery](ADR-014-security-audit-delivery.md)

Gateway decisions:

- [ADR-015 — Gateway public edge](ADR-015-gateway-public-edge.md)
- [ADR-016 — Gateway authentication context](ADR-016-gateway-auth-context.md)
- [ADR-017 — Gateway limits and failure policy](ADR-017-gateway-limits-and-failure-policy.md)

Orders decisions:

- [ADR-018 — Orders-owned orchestrated Saga](ADR-018-orders-saga.md)
- [ADR-019 — Canonical customer reference](ADR-019-orders-customer-reference.md)
- [ADR-020 — Quote authority and price snapshots](ADR-020-orders-price-snapshot.md)
- [ADR-021 — Separate command and event topics](ADR-021-orders-command-topics.md)
- [ADR-022 — Reconcile ambiguous downstream operations](ADR-022-orders-reconciliation.md)
