# Orders Event and Command Schemas

These JSON Schemas are the versioned source of truth for the initial Orders workflow. Orders workflow messages use the common command or event envelope, are delivered at least once, and are partitioned by `order_id`. The shared security-audit contract is separately versioned and keeps its own compatible audit envelope.

## Ownership

Orders publishes order facts and commands to Inventory, Payments, and Fulfillment. It consumes their fact events. A command requests work; an event records a committed fact.

## Compatibility

Within `v1`, changes are additive optional fields only. Type changes, required-field changes, semantic changes, or ownership changes require `v2` and a dual-read/dual-publish migration plan. Consumers persist Inbox uniqueness by `(consumer_group, event_id)` for facts and `(consumer_group, command_id)` for commands.

## Schemas

- Common envelopes: [`commerce.event-envelope.v1.json`](commerce.event-envelope.v1.json) and [`commerce.command-envelope.v1.json`](commerce.command-envelope.v1.json)
- Orders facts: `commerce.order.created.v1.json`, `commerce.order.cancellation_requested.v1.json`, `commerce.order.confirmed.v1.json`, `commerce.order.cancelled.v1.json`, `commerce.order.failed.v1.json`, `commerce.order.completed.v1.json`
- Inventory facts: `commerce.inventory.reserved.v1.json`, `commerce.inventory.rejected.v1.json`, `commerce.inventory.released.v1.json`
- Payment facts: `commerce.payment.authorized.v1.json`, `commerce.payment.failed.v1.json`, `commerce.payment.refunded.v1.json`, `commerce.payment.refund_failed.v1.json`
- Fulfillment facts: `commerce.fulfillment.created.v1.json`, `commerce.fulfillment.failed.v1.json`, `commerce.fulfillment.completed.v1.json`

Command schemas are `commerce.inventory.reserve.v1.json`, `commerce.inventory.release.v1.json`, `commerce.payment.authorize.v1.json`, `commerce.payment.refund.v1.json`, and `commerce.fulfillment.create.v1.json`. Their target services remain authoritative for execution and response facts; ACLs and topic policy are defined in [`docs/design/orders-kafka.md`](../../docs/design/orders-kafka.md). Inbox uniqueness uses `(consumer_group, event_id)` for events and `(consumer_group, command_id)` for commands.

Shared security audit facts are defined in [`commerce.security.audit.v1.json`](commerce.security.audit.v1.json); Identity and Orders publish only their own bounded action catalog entries.
