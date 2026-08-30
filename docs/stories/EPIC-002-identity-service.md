# EPIC-002 — Production-Ready Identity Service

## Outcome

Provide a durable Identity Service that authenticates human and machine callers, exposes public key discovery, manages sessions and explicit scopes, emits sanitized security audit events, and runs consistently in local Docker Compose and Kubernetes environments.

## Source documents

- `docs/prd/PRD-001-identity-service.md`
- `docs/architecture/identity-service.md`
- `docs/design/identity-data-model.md`
- `docs/design/identity-flows.md`
- `docs/architecture/decisions/ADR-009-authentication-model.md`
- `docs/architecture/decisions/ADR-010-rbac-and-scopes.md`
- `docs/architecture/decisions/ADR-011-asymmetric-jwt-and-jwks.md`
- `docs/architecture/decisions/ADR-012-refresh-token-rotation.md`
- `docs/architecture/decisions/ADR-013-service-to-service-identity.md`
- `contracts/openapi/identity.openapi.yaml`
- `contracts/events/commerce.security.audit.v1.json`

## Delivery order

1. STORY-006 — service foundation, configuration, and database
2. STORY-007 — registration and human login
3. STORY-008 — access tokens and JWKS
4. STORY-009 — sessions, refresh rotation, and revocation
5. STORY-010 — RBAC and user administration
6. STORY-011 — service principals and audit outbox
7. STORY-012 — password reset and email verification
8. STORY-013 — production packaging, observability, and recovery

## Epic-level acceptance

- All Identity API paths in the OpenAPI contract are implemented or explicitly disabled by configuration.
- All security audit actions in the event schema are emitted through the Identity outbox without secret material.
- Unit, integration, contract, security, race, and failure-path tests pass.
- Docker Compose starts the service with its dependencies and kind/Helm deploys it with health, logging, metrics, traces, secret references, and network boundaries.
- The service can be upgraded through compatible migrations and recovered through the documented runbooks.
