# Orders Kafka Contract

## Topics

| Topic | Direction | Owner | Partition key |
|---|---|---|---|
| `commerce.order.events.v1` | publish | Orders | `order_id` |
| `commerce.inventory.commands.v1` | publish | Orders → Inventory | `order_id` |
| `commerce.inventory.events.v1` | consume | Inventory | `order_id` |
| `commerce.payment.commands.v1` | publish | Orders → Payments | `order_id` |
| `commerce.payment.events.v1` | consume | Payments | `order_id` |
| `commerce.fulfillment.commands.v1` | publish | Orders → Fulfillment | `order_id` |
| `commerce.fulfillment.events.v1` | consume | Fulfillment | `order_id` |
| `commerce.security.audit.v1` | publish scoped Orders audit facts | Identity/Orders | bounded audit key |
| `commerce.dlq.v1` | publish | consumer owner | original order key |

Commands request work; events record committed facts. They are never mixed merely because they concern the same aggregate.

## Envelope identity matrix

| Message family | `aggregate_type` | `aggregate_id` | `data.order_id` | Response correlation |
|---|---|---|---|---|
| Orders facts | `order` | `order_id` | same as envelope ID | `saga_id` and `causation_id` link the triggering command/event |
| Inventory/Payment/Fulfillment commands | target authority (`inventory`, `payment`, `fulfillment`) | `order_id` as the target operation aggregate | required and equal to envelope ID | `operation_id` is deterministic per Saga step |
| Inventory/Payment/Fulfillment facts | target authority | authority reference when available, otherwise `order_id` | required and equal to the originating order | `operation_id` must equal the command operation ID |
| Orders audit facts | `order` | `order_id` | same as envelope ID | actor, reason, correlation, and causation are mandatory |

Consumers reject a message when the envelope identity, payload order ID, expected Saga ID, or operation ID relationship fails. JSON Schema validates shape; the Inbox/domain contract tests validate these cross-message relationships.

## Envelope

Every Orders workflow event contains `event_id`, `event_type`, `aggregate_type`, `aggregate_id`, `aggregate_version`, `saga_id`, `correlation_id`, `causation_id`, `occurred_at`, `producer`, `schema_version`, `operation_id`, and `data`. Every command contains the analogous `command_id`, `command_type`, `aggregate_version`, and `saga_id` fields. `order_id` is always the partition key for order workflow messages. Shared security-audit facts follow `commerce.security.audit.v1` and its dedicated sanitized audit envelope.

## Orders-owned messages

Publishes facts: `commerce.order.created.v1`, `commerce.order.cancellation_requested.v1`, `commerce.order.confirmed.v1`, `commerce.order.cancelled.v1`, `commerce.order.failed.v1`, and `commerce.order.completed.v1`.

Publishes commands: `commerce.inventory.reserve.v1`, `commerce.inventory.release.v1`, `commerce.payment.authorize.v1`, `commerce.payment.refund.v1`, and `commerce.fulfillment.create.v1` as message types on the explicit command topics.

Consumes facts: inventory reserved/rejected/released; payment authorized/failed/refunded/refund_failed; fulfillment created/failed/completed. Their schemas are versioned under `contracts/events/`.

Orders does not publish notification commands in v1. Notifications subscribes to the Orders fact stream and owns delivery attempts, templates, provider credentials, and delivery status. A notification failure never rolls back or blocks the order Saga. The `commerce.notification.commands.v1` topic remains a Notifications-owned integration boundary for future producer-specific use; it is not an Orders dependency.

Orders publishes scoped audit facts (`orders.cancellation.requested`, `orders.privileged_read`, `orders.repair.executed`, `orders.dlq.replayed`, and `orders.compensation.intervened`) to the shared security-audit topic through its own transactional Outbox. Orders audit records require actor, reason, correlation, causation, and outcome in the application contract; the common schema keeps nullable fields for Identity compatibility. The schema binds `producer=orders` and `aggregate_id=ord_...`; Identity remains the owner of identity actions. Orders must not place secrets or customer payloads in audit records.

## Principal and ACL matrix

The HTTP API and relay are separate runtime principals even when they share the same PostgreSQL database:

| Principal | Kafka permissions | Database permissions |
|---|---|---|
| `svc-orders-api` | none | read/write Orders transaction tables; no migration ownership |
| `svc-orders-relay` | publish Orders facts/commands; consume Inventory/Payments/Fulfillment facts; publish DLQ and scoped audit facts | read/lease Outbox, write Inbox/Saga/audit state |
| `svc-orders-replay` | consume only approved replay partitions; publish no new business command without an audited operation | read diagnostic state; invoke only the repair application port |

Kafka ACLs bind these principals to TLS client identities. The `producer` field is checked against the authenticated principal and topic ACL; schema validation alone is never treated as message authenticity. `svc-orders-api` does not receive Kafka credentials, so an API compromise cannot publish arbitrary commands.

Envelope producers map to authenticated runtime principals as follows: `orders` → `svc-orders-relay`, `inventory` → `svc-inventory-relay`, `payments` → `svc-payments-relay`, `fulfillment` → `svc-fulfillment-relay`, and `identity` → `svc-identity-relay`. Each producer may publish only its owned fact or command topic; Orders consumes dependency facts through an ACL granting read access only to `svc-orders-relay`.

## Provisioning baseline

The initial topology uses one partition locally and six partitions in staging/production for each order-workflow topic. Staging and production use replication factor 3 with `min.insync.replicas=2`; local uses replication factor 1. Order facts retain 30 days, commands 7 days, and DLQ records 90 days unless a platform retention decision is stricter. Audit retention follows the platform audit policy.

`orders-saga-v1` is the normal consumer group and `orders-replay-v1` is the controlled replay group. Transient delivery retries are durable and capped at 8 attempts with delays of 1s, 5s, 30s, 2m, 5m, 10m, 15m, and 30m. Exhaustion or permanent validation failure writes the original message identity to DLQ; replay preserves that identity and requires an audited operation.

## Delivery and compatibility

Delivery is at-least-once. Consumer groups are exactly `orders-saga-v1` and `orders-replay-v1` for controlled operations. Inbox uniqueness is `(consumer_group, message_id)`, using `event_id` for events and `command_id` for commands. Unknown schema versions, invalid producer, wrong aggregate, invalid IDs, missing correlation/causation, and impossible transitions are quarantined.

Schemas are backward-compatible within `v1`: additive optional fields only, no type or semantic changes. Incompatible changes require `v2`, dual-read/dual-publish migration, and a documented cutover. Kafka uses TLS, authenticated identities, topic ACLs, retention, and environment isolation.

## Retry and DLQ

Transient errors use bounded durable retry with backoff. Permanent schema/auth/domain errors go directly to DLQ. A DLQ record includes original topic, partition, offset, message ID, type, consumer group, attempts, sanitized reason, and trace/correlation IDs. Replay reuses the original event ID and Inbox protection.
