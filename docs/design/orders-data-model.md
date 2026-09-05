# Orders Data Model

## PostgreSQL ownership

Orders owns `orders_db` and the `orders_user` role. No application query crosses into another service database. Migrations use expand/migrate/contract sequencing; destructive rollback is not the deployment strategy.

## Tables

### `orders`

`id`, `customer_id`, `currency`, `total_minor`, `status`, `failure_code`, `reservation_id`, `payment_id`, `shipment_id`, `aggregate_version`, `created_at`, `updated_at`.

Constraints: unique ID, supported uppercase currency, non-negative bounded totals when finalized, positive aggregate version, and indexed `(customer_id, created_at, id)` for stable listing. `total_minor` is nullable only while the initial Inventory quote is pending.

`reservation_id`, `payment_id`, and `shipment_id` are opaque references owned by the corresponding service. Orders never dereferences them through another service database. A future shipping-address snapshot, if required by the approved contract, is encrypted at rest and redacted from logs; the v1 create contract intentionally does not accept address data.

### `order_items`

`order_id`, `line_number`, `product_id`, `quantity`, `unit_price_minor`, `line_total_minor`, `quote_id`, `quote_version`.

Primary key `(order_id, line_number)` and foreign key to `orders`. Prices and quote metadata are immutable after the Inventory quote is accepted; item intent remains price-less only during the initial `PENDING` quote phase.

### `saga_instances` and `saga_steps`

The instance stores `saga_id`, `order_id`, current phase, version, timestamps, and failure class. Steps store operation ID, target service, command type, status, attempts, next attempt, completed time, and sanitized last error. Unique `(order_id)` and `(saga_id, operation_id)` prevent duplicate work.

### `inbox_messages`

`consumer_group`, `message_id`, `message_type`, `payload_hash`, `received_at`, `processed_at`, `status`, `attempts`, `next_attempt_at`, `last_error`, `correlation_id`, `causation_id`. Unique `(consumer_group, message_id)`, where an event's `message_id` is its `event_id` and a command's `message_id` is its `command_id`.

### `outbox_events`

`id`, `topic`, `partition_key`, `message_type`, `aggregate_type`, `aggregate_id`, `payload`, `headers`, `created_at`, `published_at`, `attempts`, `next_attempt_at`, `lease_owner`, `lease_until`, `last_error`. Index unpublished rows by `(published_at, next_attempt_at, created_at)`.

### `orders_audit`

Orders-owned privileged actions (`orders.privileged_read`, cancellation requests, repair, DLQ replay, and compensation intervention) are stored as sanitized audit rows before their `commerce.security.audit.v1` Outbox event. The row stores actor class/reference, action, target order, reason code, correlation ID, and timestamp only; it never stores tokens, request bodies, payment data, or address data.

### `idempotency_records`

`actor_id`, `operation`, `key`, `request_hash`, `status`, `resource_id`, `response_status`, `response_body`, `created_at`, `expires_at`. Unique `(actor_id, operation, key)`; store only bounded, non-sensitive response data.

## Transaction boundaries

Create transaction: validate request → insert price-less order intent/items → insert Saga → insert idempotency record → insert `OrderCreated` Outbox → commit. Inventory quote/reservation and price snapshot are applied in the next Inbox transaction.

Message transaction: insert Inbox → lock/check order version and Saga step → apply domain transition → insert resulting Outbox/step changes → mark Inbox processed → commit.

Cancellation transaction follows the same pattern with its own idempotency operation and `OrderCancellationRequested` Outbox event. If the optional body is omitted, the application records the stable reason code `CUSTOMER_REQUESTED`.

## Retention

Retention is configurable per environment and must cover the maximum client retry window, event replay window, audit obligations, and reconciliation period. Purging requires metrics and an operator-reviewed policy; it must not remove active idempotency, Inbox, Outbox, or Saga records.
