---
title: Orders Service
status: final
created: 2026-09-05
updated: 2026-09-05
---

# PRD: Orders Service

## 0. Purpose and sources

This is the documentation-complete product and behavior contract for the StreamWeave Orders Service. It is derived from `01_event_driven_ecommerce_order_management_platform.md`, `docs/brief.md`, `docs/project_context.md`, the Gateway contract, and the approved Identity authorization model. Implementation stories may begin only after the dependency gates listed in this document are accepted.

Orders is a business service, not a generic workflow engine. It owns orders and the order Saga while Inventory, Payments, Fulfillment, Notifications, and Identity remain independent authorities.

## 1. Outcome

Allow an authenticated customer or explicitly authorized operator to create, inspect, list, and cancel orders while honestly exposing asynchronous progress and remaining correct under duplicates, retries, timeouts, partial failure, and recovery.

## 2. Scope

### In scope

- Order creation, retrieval, cursor-paginated listing, and authorized cancellation.
- Immutable item and price snapshots using integer minor units and ISO currency.
- Durable order lifecycle and Saga state.
- Transactional Outbox, durable Inbox, HTTP idempotency, retry, DLQ, and replay controls.
- Versioned REST and Kafka command/event contracts.
- Structured logs, low-cardinality metrics, traces, dashboards, alerts, runbooks, and recovery procedures.

### Out of scope

- Product or stock authority, payment processing, shipment authority, notification delivery, or user credentials.
- Real card data, PCI scope, real customer email, frontend work, GraphQL, event sourcing everywhere, and cross-service database access.
- A second public API origin or a generic proxy.

## 3. Actors and journeys

- **Customer:** creates an order for themselves, observes its status, and requests cancellation while eligible.
- **Operator:** uses explicit `orders:read:any`, `orders:write:any`, or `orders:cancel:any` permissions; Orders still applies business-state rules.
- **Gateway:** authenticates the public request and forwards a trusted service context; it does not own order authorization or idempotency.
- **Platform operator:** diagnoses lag, failed Sagas, Outbox/DLQ backlog, and database recovery without seeing unnecessary PII.

### Primary journey

1. The customer submits products and quantities with an `Idempotency-Key`.
2. Orders validates the actor, request bounds, and currency, then commits a price-less `PENDING` order intent plus Saga, idempotency record, and `OrderCreated` Outbox event. The authoritative Inventory quote is applied asynchronously before payment authorization.
3. The API returns `202 Accepted`; no downstream completion is implied.
4. The Saga requests inventory reservation, payment authorization, and fulfillment in order.
5. The customer reads the current public status. Failures trigger bounded, observable, idempotent compensation.

## 4. Functional requirements

### FR-ORD-001 — Create order

Given a trusted actor submits at least one valid item, Orders shall validate authorization and bounds, persist a `PENDING` order intent, persist Saga state, persist the idempotency result, and write `commerce.order.created.v1` to the Outbox in one PostgreSQL transaction. The first Saga step asks Inventory to reserve stock and return the authoritative quote; only then may Orders snapshot prices and authorize payment. It shall return `202` only after the local commit.

### FR-ORD-002 — Durable idempotency

The same actor, operation, and key with the same canonical request shall replay the original `202` result. The same key with a different request fingerprint shall return `409`. Concurrent first requests shall create at most one order. Redis failure shall not change correctness.

### FR-ORD-003 — Read and list orders

Customers shall only read and list their own orders. Operators require explicit `orders:read:any`. Listing shall use stable, bounded, opaque cursors bound to actor, scope, filters, and sort. Unauthorized resource access shall use the repository's non-enumerating policy: `404` for a resource outside the actor's visible set.

### FR-ORD-004 — Request cancellation

An authorized actor may request cancellation only while the documented state and fulfillment rules allow it. The mutation is idempotent, returns `202` with `CANCEL_REQUESTED` after local commit, and starts durable compensation. Ineligible cancellation returns `409` without changing the order.

### FR-ORD-005 — Orchestrate the Saga

Orders shall issue durable commands and consume versioned events. Every transition is guarded by current aggregate version, Saga step, operation ID, and Inbox deduplication. Late, duplicate, or replayed events shall never reopen terminal state or repeat a financial side effect.

### FR-ORD-006 — Recover safely

Transient failures receive bounded durable retry. Poison messages are quarantined with original metadata and require authorized replay. Compensation failures remain visible as `FAILED_REQUIRES_REVIEW` internal Saga state until repaired.

## 5. Public status model

The public model is intentionally smaller than internal Saga state:

`PENDING` → `CONFIRMED` → `COMPLETED`

`PENDING` or `CONFIRMED` → `CANCEL_REQUESTED` → `CANCELLED`

Any non-terminal state may reach `FAILED` after a documented failure policy. `COMPLETED`, `CANCELLED`, and terminal `FAILED` are immutable. Internal phases and compensation state are not exposed as public status values.

## 6. Non-functional requirements

- API availability target: 99.9% monthly for healthy dependencies.
- Mutation response p95: ≤ 500 ms after local commit path is available.
- Read p95: ≤ 300 ms; list p95: ≤ 500 ms for the maximum bounded page.
- Outbox publication p95: ≤ 10 seconds; alert at 60 seconds.
- No committed order is lost (RPO 0 for committed local transactions).
- Database recovery target: RTO ≤ 30 minutes.
- At least 85% application coverage; race, contract, integration, security, Compose, deployment, and recovery checks are mandatory.
- No raw credentials, bearer tokens, payment data, full addresses, or unbounded user values in logs, metrics, traces, events, or errors.

## 7. Contract and ownership impact

- REST: `contracts/openapi/orders.openapi.yaml`; Gateway facade must use the same `usr_` customer reference and public statuses.
- Events and commands: versioned JSON Schemas under `contracts/events/`; Kafka ownership and compatibility are in `docs/design/orders-kafka.md`.
- Data: `orders_db` only; Orders owns orders, items, Saga, Inbox, Outbox, and idempotency records.
- Operations: centralized monitoring under `monitoring/`; Orders adds panels and alerts to the shared stack.

## 8. Security requirements

Orders shall authenticate the Gateway service principal, validate audience/issuer/expiry/scopes, bind actor context to the requested customer, enforce explicit resource permissions, minimize PII, reject client prices, use TLS/authenticated Kafka, and prevent replay or duplicate side effects through Inbox and operation IDs.

## 9. Acceptance criteria

- [ ] Every documented route has request, response, authorization, idempotency, error, and observability behavior.
- [ ] A valid create request commits order, Saga, idempotency, and Outbox atomically and returns `202`.
- [ ] Same-key replay returns the original result; conflicting reuse returns `409`; concurrent creates produce one order.
- [ ] Ownership and operator permissions prevent IDOR and customer enumeration.
- [ ] Illegal transitions, stale events, duplicate events, duplicate commands, and replayed financial events are harmless.
- [ ] Inventory, payment, and fulfillment failure paths have bounded retries and idempotent compensation.
- [ ] Outbox, Inbox, DLQ, Saga repair, database restore, and Kafka recovery have runbooks and tests.
- [ ] Docker, local Compose, staging/production Helm contracts, dashboards, alerts, and 85% coverage gates are defined before implementation.

## 10. Decisions closed by this PRD

- `usr_...` Identity subject is the canonical v1 customer reference; no hidden `cus_` mapping is introduced.
- Inventory is the initial authority for product availability and price quotes; its asynchronous reservation result carries a versioned, authenticated quote that Orders snapshots before payment. A dedicated Pricing service is deferred.
- `PENDING` is the only initial acceptance status; `ACCEPTED` is removed from the service contract.
- Orders owns an orchestrated Saga and explicit command topics; commands are not mixed into domain-event topics.
- Notification delivery is an observable side effect and does not block order confirmation or fulfillment.
- Compensation failure is durable, alertable, and requires operator repair; it is never silently swallowed.

## 11. Deferred decisions

- Exact customer-facing cancellation window after fulfillment dispatch, to be finalized when Fulfillment is documented; the current contract returns `409` unless Fulfillment explicitly reports the shipment cancellable.
- Inventory must implement the Orders quote contract: detached JWS, public-key rotation, expiry, deterministic reservation operation IDs, and explicit timeout/rejection facts. This is a gate for Orders integration, not for foundation/domain work.
- Capacity-derived SLO refinement after load evidence.
- Multi-tenant partitioning, if the product later adds tenants.

## 12. Implementation sequencing gate

Foundation, domain, API, persistence, Inbox/Outbox, contract validation, observability, and deployment scaffolding may be implemented on Orders now. The end-to-end Saga stories that call Inventory, Payments, or Fulfillment may not be accepted until the dependency-owned request/result and reconciliation contracts are versioned and covered by executable fixtures. This sequencing keeps the Orders core testable without inventing a generic downstream client.
