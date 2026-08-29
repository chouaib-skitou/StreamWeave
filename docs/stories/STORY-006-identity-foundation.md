# STORY-006 — Establish Identity Service Foundation

## User / system outcome

As a platform operator, I can start Identity with validated configuration, database migrations, health endpoints, and graceful shutdown.

## Context

Identity must be independently deployable and must fail safely when PostgreSQL, Redis, telemetry, or signing-key configuration is unavailable.

## Acceptance criteria

- [ ] Given valid configuration, when the service starts, then it serves `/health/live`, `/health/ready`, and `/health/startup`.
- [ ] Given missing database or signing-key configuration, when the service starts, then readiness fails and no token is issued.
- [ ] Given a fresh `identity_db`, when migrations run, then all tables and seed permissions are created idempotently.
- [ ] Given shutdown, when the process receives cancellation, then HTTP, database, relay, and telemetry resources close within the configured deadline.
- [ ] Given domain tests, when they run without infrastructure, then no domain package imports an infrastructure dependency.

## API changes

Health endpoints and the error envelope are introduced; Identity routes follow the OpenAPI contract.

## Event changes

No published event beyond migration/boot observability.

## Data changes

Create the Identity tables defined in `docs/design/identity-data-model.md`.

## Failure cases

Configuration validation, migration failure, database outage, Redis outage, telemetry exporter timeout, and shutdown timeout must be explicit and observable.

## Observability

Emit startup/readiness metrics and structured lifecycle logs without configuration secrets.

## Security considerations

Reject insecure production defaults, never print environment values, and run the container as non-root.

## Test plan

Unit configuration tests, migration integration tests, health handler tests, shutdown tests, and architecture dependency checks.

## Out of scope

Business authentication flows and Kubernetes resource implementation are delivered by later stories.

## Dependencies

`docs/prd/PRD-001-identity-service.md`, `docs/architecture/identity-service.md`, and `docs/design/identity-data-model.md`.

## Validation evidence

Record command output and environment details in the story after implementation.
