# Gateway Service

Status: implemented on `feat/gateway-service`; merge to `dev` and `main` remains governed by the repository PR flow.

The Gateway is StreamWeave's explicit public HTTP edge. It owns request IDs, trace propagation, JWT/JWKS verification, coarse scopes, Redis rate limits, routing, standardized errors, and access logs. It does not own users, sessions, orders, business authorization, durable state, or a database.

Local access is `http://localhost:8081`; Identity already uses `http://localhost:8080`. The Gateway container still listens on `:8080`, and Kubernetes gives each service a separate ClusterIP.

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

The implementation follows the accepted documentation gate, the repository's Clean / Hexagonal rules, and the service-specific instructions in `AGENTS.md`.

## Local runtime

The local Compose stack maps Gateway to `http://localhost:8081` and expects Identity to be available on `http://localhost:8080`. Copy `deploy/gateway/local/.env.example` to `.env`, replace the development service token, and run `make gateway-compose-up`. The Gateway container listens on `:8080`; the host mapping is deliberately `8081` to avoid Identity's `8080` mapping.

Use `make gateway-test` for race-enabled tests and the 85% application coverage gate, `make gateway-lint` for vet/format checks, and `make gateway-kind-deploy` for Helm installation. Staging and production use an external Secret containing `GATEWAY_SERVICE_CLIENT_SECRET`; static tokens are rejected outside development.
