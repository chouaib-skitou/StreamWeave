---
name: Orders Service
type: architecture-spine
purpose: build-substrate
altitude: feature
paradigm: Clean / Hexagonal Architecture
scope: Order aggregate, asynchronous lifecycle, durable Saga, REST API, Kafka integration, and recovery
status: documentation-complete
created: 2026-09-05
updated: 2026-09-05
sources:
  - 01_event_driven_ecommerce_order_management_platform.md
  - docs/brief.md
  - docs/project_context.md
  - docs/prd/PRD-003-orders-service.md
  - docs/architecture/gateway-service.md
  - docs/architecture/identity-service.md
companions:
  - contracts/openapi/orders.openapi.yaml
  - docs/design/orders-state-machine.md
  - docs/design/orders-saga.md
  - docs/design/orders-reconciliation.md
  - docs/design/orders-kafka.md
---

# Orders Service Architecture

## Paradigm and runtime shape

Orders is an independently deployable Go service with two workloads sharing the same application core: an HTTP API and an Outbox/Saga relay. Both use the same `orders_db` under least-privilege database roles, but only the relay receives Kafka credentials; each workload has an explicit lifecycle, health behavior, service account, and consumer group.

```text
HTTP / Kafka inbound adapters
          ↓
Application commands, queries, and message handlers
          ↓
Order aggregate, state machine, and Saga policy
          ↓
Ports: transaction, repositories, idempotency, clock, event/command delivery
          ↓
PostgreSQL, Kafka, Redis limiter/cache, telemetry adapters
```

Domain code must not import HTTP, pgx, Kafka, Redis, Kubernetes, or OpenTelemetry packages. Interfaces live beside the consuming application boundary and describe real capabilities only.

## Ownership boundaries

| Concern | Owner | Orders behavior |
|---|---|---|
| Users, sessions, roles, JWT keys | Identity | Validate trusted actor context; never query Identity DB |
| Public routing and coarse scope gate | Gateway | Receive only authenticated Gateway calls |
| Order aggregate and public status | Orders | Authoritative `orders_db` state |
| Product, availability, and quote | Inventory | Consume versioned quote/reservation results; snapshot price |
| Authorization and refund authority | Payments | Orders requests commands; never stores payment credentials |
| Shipment lifecycle | Fulfillment | Orders stores external reference and projected status only |
| Notification delivery | Notifications | Side effect; never blocks order truth |
| Rate-limit acceleration | Redis | Disposable; never source of truth |

Orders may store external references (`product_id`, `reservation_id`, `payment_id`, `shipment_id`) but never replicated authority or cross-service joins.

## Invariants

- One order has one immutable external ID, one customer reference, one currency, and at least one item.
- Money is integer minor units; quantities, item count, and total are bounded before arithmetic and overflow is rejected.
- Price snapshots are immutable after acceptance and are traceable to a quote ID/version.
- Every durable mutation that emits a message writes the state and Outbox record in one local transaction.
- Every consumed message is Inbox-deduplicated and every downstream command has a deterministic operation ID.
- Aggregate version and Saga step guard transitions; terminal public states cannot regress.
- A timeout after a remote success is resolved by event reconciliation, never blind re-execution.

## Consistency

PostgreSQL provides strong local transaction boundaries. Kafka workflows are eventually consistent. `202 Accepted` means only that Orders committed the local order and its first Outbox message. The API exposes `PENDING`, `CANCEL_REQUESTED`, and terminal progress honestly.

## Deployment invariants

Orders runs non-root with a read-only filesystem, dropped Linux capabilities, pinned build inputs, resource limits, startup/liveness/readiness probes, a PDB, and a restrictive NetworkPolicy. It has no public ingress. Egress is limited to `orders_db`, Kafka, Redis when configured, Inventory/Payments/Fulfillment internal endpoints when required, and telemetry. Monitoring is shared at platform level.

## Architecture decisions

- **ORD-AD-01:** Orders owns an orchestrated Saga because it is the only service with the complete order workflow view.
- **ORD-AD-02:** Commands use explicit command topics; fact events use domain topics. A command requests work; an event records committed fact.
- **ORD-AD-03:** Durable idempotency, Inbox, Outbox, and Saga state are PostgreSQL-owned and transactionally coordinated.
- **ORD-AD-04:** Identity `sub` (`usr_...`) is the canonical customer reference for v1.
- **ORD-AD-05:** Inventory supplies authoritative price/availability quotes; Orders snapshots them and Payments receives the resulting amount/currency.
- **ORD-AD-06:** Public status values are stable and intentionally smaller than internal Saga phases.

## Deferred

The final cancellation rule after shipment dispatch and capacity-tuned SLOs remain owned by future service documents. Inventory must implement the quote contract in `orders-pricing-boundary.md` before the Orders integration story is accepted. None changes the Orders ownership or transaction invariants above.
