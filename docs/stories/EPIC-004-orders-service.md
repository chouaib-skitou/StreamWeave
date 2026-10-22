# EPIC-004 — Orders Service

## Outcome

Provide an independently deployable, secure, observable Orders service that owns the order aggregate and orchestrates its asynchronous lifecycle without sharing databases or pretending distributed work is synchronous.

## Scope

- Order creation, retrieval, listing, and cancellation.
- Durable aggregate, Saga, Inbox, Outbox, and HTTP idempotency.
- Inventory, Payment, Fulfillment, and Notification contracts.
- Kafka retry/DLQ/replay, operational telemetry, Docker, Compose, Helm, and recovery runbooks.

## Stories

- [STORY-019](STORY-019-orders-foundation.md) — Service foundation and architecture boundaries
- [STORY-020](STORY-020-orders-domain-and-state-machine.md) — Order aggregate, money, and lifecycle invariants
- [STORY-021](STORY-021-orders-api-and-authorization.md) — REST contract, ownership, and operator authorization
- [STORY-022](STORY-022-orders-persistence-and-idempotency.md) — PostgreSQL model and durable idempotency
- [STORY-023](STORY-023-orders-outbox-inbox.md) — Transactional Outbox and Inbox processing
- [STORY-024](STORY-024-orders-saga-happy-path.md) — Inventory/payment/fulfillment happy path
- [STORY-025](STORY-025-orders-compensation.md) — Cancellation and failure compensation
- [STORY-026](STORY-026-orders-kafka-contracts.md) — Versioned commands, events, ACLs, and DLQ
- [STORY-027](STORY-027-orders-observability.md) — Metrics, traces, logs, dashboards, and alerts
- [STORY-028](STORY-028-orders-deployment.md) — Docker, local Compose, staging, and production Helm
- [STORY-029](STORY-029-orders-recovery-and-validation.md) — Recovery, security, E2E, chaos, and release gates

## Dependencies

Gateway and Identity contracts are available. Inventory, Payments, Fulfillment, and Notifications must finalize their counterpart schemas before their consumers are implemented. Shared Kafka, PostgreSQL, Redis, and monitoring conventions are platform-owned.

## Definition of done

All stories pass unit, integration, contract, security, race, Compose, deployment, recovery, and E2E checks; application coverage is at least 85%; no direct database access crosses a service boundary; documented API/event/data/runbook artifacts match the implementation.
