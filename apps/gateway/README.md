# Gateway Service

Status: documentation-ready; implementation intentionally not started.

The Gateway is StreamWeave's explicit public HTTP edge. It owns request IDs, trace propagation, JWT/JWKS verification, coarse scopes, Redis rate limits, routing, standardized errors, and access logs. It does not own users, sessions, orders, business authorization, durable state, or a database.

## Documentation

- [PRD](../../docs/prd/PRD-002-gateway-service.md)
- [Architecture](../../docs/architecture/gateway-service.md)
- [Route matrix](../../docs/design/gateway-route-matrix.md)
- [Interaction flows](../../docs/design/gateway-flows.md)
- [Authentication](../../docs/design/gateway-authentication.md)
- [Resilience](../../docs/design/gateway-resilience.md)
- [Deployment](../../docs/design/gateway-deployment.md)
- [Observability](../../docs/design/gateway-observability.md)
- [OpenAPI contract](../../contracts/openapi/gateway.openapi.yaml)
- [Stories](../../docs/stories/EPIC-003-gateway-service.md)

Do not add Gateway code until the documentation gate is accepted. New implementation must follow the repository's Clean / Hexagonal rules and the service-specific instructions in `AGENTS.md`.
