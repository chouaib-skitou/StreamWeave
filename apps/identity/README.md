# Identity Service

The Identity Service owns human users, credentials, sessions, roles, refresh-token rotation, service principals, JWKS publication, and security audit events for the platform.

## Documentation

- Requirements: [`../../docs/prd/PRD-001-identity-service.md`](../../docs/prd/PRD-001-identity-service.md)
- Architecture: [`../../docs/architecture/identity-service.md`](../../docs/architecture/identity-service.md)
- Data model: [`../../docs/design/identity-data-model.md`](../../docs/design/identity-data-model.md)
- Flows: [`../../docs/design/identity-flows.md`](../../docs/design/identity-flows.md)
- Deployment: [`../../docs/design/identity-deployment.md`](../../docs/design/identity-deployment.md)
- API contract: [`../../contracts/openapi/identity.openapi.yaml`](../../contracts/openapi/identity.openapi.yaml)
- Audit event: [`../../contracts/events/commerce.security.audit.v1.json`](../../contracts/events/commerce.security.audit.v1.json)
- Stories: [`../../docs/stories/EPIC-002-identity-service.md`](../../docs/stories/EPIC-002-identity-service.md)

## Implementation status

The production-like service implementation is available on the Identity feature branch. It includes the Go application, sqlc-generated PostgreSQL adapter, transactional outbox relay, Redis rate limiting and revocation, Ed25519/JWKS, Docker Compose, Helm deployment, health endpoints, metrics, and automated verification targets.

## Security boundary

Never commit passwords, tokens, private keys, credentials, real payment data, or local secret files. The service-token endpoint is internal-only and must not be exposed through the public gateway.
