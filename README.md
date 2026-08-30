# Event-Driven E-commerce Order Management Platform

Portfolio-grade event-driven commerce platform in Go. The project demonstrates service boundaries, hexagonal architecture, versioned REST and Kafka contracts, database-per-service ownership, transactional outbox, idempotent consumers, Saga coordination, observability, and reliable recovery from partial failure.

## Project status

The platform is being built incrementally. The Identity Service is implemented
on its feature branch with its contracts, persistence, local observability
stack, Docker image workflow, Helm chart, and validation evidence. The remaining
business services are still planned and will be delivered through reviewed
feature branches.

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
