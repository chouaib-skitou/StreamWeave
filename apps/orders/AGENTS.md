# Orders Service Agent Guide

## Responsibility

Orders owns the order aggregate, customer-facing order queries, durable HTTP idempotency, the order Saga, Inbox processing, and the Transactional Outbox. It is the only owner of `orders_db`.

## Architecture rules

- Keep domain invariants independent from HTTP, PostgreSQL, Kafka, Redis, Kubernetes, and OpenTelemetry.
- Follow Clean / Hexagonal Architecture: inbound adapters -> application -> domain; outbound adapters implement ports.
- Keep repositories and ports specific to their consumer. Never add a generic repository abstraction.
- Do not query or write Identity, Inventory, Payments, Fulfillment, or Notifications databases.
- Redis is disposable acceleration only. PostgreSQL is authoritative for orders, Saga state, Inbox, Outbox, and idempotency.
- Use integer minor units plus an ISO-4217 currency code for money. Never use floating-point money.

## Ownership and authorization

- Gateway is the only public edge. Orders must authenticate the Gateway service principal before trusting actor context.
- Orders performs the final ownership and business authorization checks; a valid JWT or Gateway scope is not sufficient by itself.
- The canonical customer reference is the Identity subject (`usr_...`) for v1. A client-supplied `customer_id` must match the trusted actor unless an explicit `orders:write:any` permission applies.
- Never trust client-controlled `X-StreamWeave-*` headers, emails, roles, or prices.

## Domain invariants

- An order contains at least one item.
- Item quantities, item count, total amount, and currency are bounded and validated.
- Prices are authoritative snapshots obtained through the documented Inventory quote boundary, including expiry and detached-JWS verification; clients never supply prices or totals.
- Creation persists the order, Saga state, idempotency record, and `OrderCreated` Outbox message atomically.
- Public statuses are `PENDING`, `CONFIRMED`, `CANCEL_REQUESTED`, `CANCELLED`, `COMPLETED`, and `FAILED`.
- Terminal statuses cannot regress. Confirmation requires successful inventory reservation and payment authorization.
- Every mutation, downstream command, consumed event, and compensation operation is idempotent.

## Messaging rules

- Kafka delivery is at-least-once. Consumers must deduplicate by `(consumer_group, message_id)` in a durable Inbox; `message_id` maps to `event_id` for facts and `command_id` for commands.
- Partition every order-related command and event by `order_id`.
- Validate producer, schema version, aggregate identity, correlation, causation, and transition legality.
- Persist state transitions and resulting Outbox messages in one local transaction.
- Retry transient failures with bounded durable policy; quarantine poison messages in the DLQ and require authorized replay.
- Never propagate bearer tokens or payment credentials in events.

## Technology baseline

Follow the blueprint baseline for Orders: Go 1.26.6, `pgx/v5`, `golang-migrate`, `franz-go`, PostgreSQL 18.4, and Kafka 4.3.1. Any repository-wide dependency upgrade required to support this baseline is a separate platform change and must be completed before the corresponding integration story.

## HTTP contract

- Implement only `contracts/openapi/orders.openapi.yaml`.
- `POST /v1/orders` and `POST /v1/orders/{order_id}/cancel` require `Idempotency-Key` and return `202` after the local transaction commits.
- Query endpoints use bounded opaque cursor pagination and enforce ownership before returning data.
- Return RFC 9457 Problem Details with a request ID; do not reveal whether another customer's order exists.

## Production gate

Maintain at least 85% application coverage and pass formatting, static analysis, race-enabled unit tests, PostgreSQL/Kafka integration tests, contract tests, security tests, Compose smoke tests, deployment validation, and documented failure/recovery tests.

Never commit secrets, `.env` files, tokens, private keys, payment data, generated code, or local BMAD artifacts. Update the relevant docs and contracts before implementation changes.
