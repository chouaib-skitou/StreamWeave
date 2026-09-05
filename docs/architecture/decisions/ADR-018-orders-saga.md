# ADR-018: Orders-owned orchestrated Saga

- **Status:** accepted
- **Date:** 2026-09-05
- **Owners:** Platform architecture and Orders

## Context

Order creation crosses Inventory, Payments, Fulfillment, and Notifications. Kafka is at-least-once and dependencies can fail after accepting work. A workflow without a durable owner would make compensation, visibility, and repair ambiguous.

## Decision

Orders owns a durable orchestrated Saga in `orders_db`. It emits explicit commands on domain-specific command topics and reacts to versioned fact events. Each step has a deterministic operation ID, timeout/retry policy, Inbox deduplication, and a persisted compensation state. Public order status is projected separately from internal Saga phase.

Happy path:

```text
create order
 → reserve inventory
 → authorize payment
 → confirm order
 → create shipment
 → complete order
```

Failure path:

```text
inventory rejection → FAILED
payment failure → release inventory → FAILED
shipment failure → bounded retry → refund payment → release inventory → FAILED_REQUIRES_REVIEW if repair fails
```

## Consequences

- Workflow ownership, progress, and repair are observable in one durable boundary.
- Compensation is explicit and can be retried safely.
- Orders depends on message contracts and must tolerate delayed, duplicate, and out-of-order delivery.
- The Saga is not exactly-once; business effects are made idempotent.

## Rejected alternatives

- **Pure choreography:** rejected for the initial workflow because compensation and operator visibility would be scattered across services.
- **Distributed transaction:** rejected because services own separate databases and must remain independently deployable.
- **Redis-only workflow state:** rejected because Redis is disposable.
