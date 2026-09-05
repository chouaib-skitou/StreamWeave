# StreamWeave — Event-Driven Commerce Platform

StreamWeave is a portfolio-grade event-driven commerce platform in Go. It demonstrates service boundaries, hexagonal architecture, versioned REST and Kafka contracts, database-per-service ownership, transactional outbox, idempotent consumers, Saga coordination, observability, and reliable recovery from partial failure.

## Project status

The platform is being built incrementally. Identity and Gateway are implemented
with their contracts, local observability stack, Docker image workflows, Helm
charts, and validation evidence. Orders now has a documentation-complete
package; its service code and the remaining business services follow through
reviewed feature branches.

## Documentation map

- [Project brief](docs/brief.md)
- [Project context](docs/project_context.md)
- [Architecture](docs/architecture/README.md)
- [Architecture decisions](docs/architecture/decisions/README.md)
- [Product requirements](docs/prd/README.md)
- [Stories](docs/stories/README.md)
- [Design](docs/design/README.md)
- [Runbooks](docs/runbooks/README.md)
- [Contracts](contracts/README.md)
- [Identity Service PRD](docs/prd/PRD-001-identity-service.md)
- [Identity Service architecture](docs/architecture/identity-service.md)
- [Gateway Service PRD](docs/prd/PRD-002-gateway-service.md)
- [Gateway Service architecture](docs/architecture/gateway-service.md)
- [Gateway Service API contract](contracts/openapi/gateway.openapi.yaml)
- [Orders Service PRD](docs/prd/PRD-003-orders-service.md)
- [Orders Service architecture](docs/architecture/orders-service.md)
- [Orders Service API contract](contracts/openapi/orders.openapi.yaml)
- [Orders Service documentation gate](docs/bmad/implementation-artifacts/spec-orders-service-documentation.md)

Private BMAD tooling, agent instructions, planning scratch work, and the original blueprint stay local and are excluded by `.gitignore`.

## Operating principles

- Each business service owns its database.
- REST is used at client and query boundaries; Kafka is used for asynchronous domain integration.
- Durable state changes that publish events use a transactional outbox.
- Consumers are idempotent and tolerate duplicate delivery.
- Redis is never the source of truth for orders, payments, inventory, or Saga state.
- Money is represented as integer minor units and timestamps are UTC.

## Branching

`main` and `dev` are protected. Changes are delivered through short-lived feature, fix, docs, or chore branches and reviewed pull requests.

See [CONTRIBUTING.md](CONTRIBUTING.md) and [PULL_REQUEST_RULES.md](PULL_REQUEST_RULES.md).
