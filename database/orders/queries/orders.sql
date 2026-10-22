-- name: GetOrder :one
SELECT id, customer_id, currency, total_minor, status, failure_code, reservation_id,
       payment_id, shipment_id, inventory_released, payment_refunded,
       aggregate_version, created_at, updated_at
FROM orders WHERE id = $1;

-- name: GetOrderItems :many
SELECT order_id, line_number, product_id, quantity, unit_price_minor, line_total_minor,
       quote_id, quote_version
FROM order_items WHERE order_id = $1 ORDER BY line_number;

-- name: ListOrders :many
SELECT id, customer_id, currency, total_minor, status, failure_code, reservation_id,
       payment_id, shipment_id, inventory_released, payment_refunded,
       aggregate_version, created_at, updated_at
FROM orders
WHERE customer_id = $1
  AND ($2::text = '' OR status = $2)
  AND (sqlc.narg('cursor_created_at')::timestamptz IS NULL
       OR (created_at, id) < (sqlc.narg('cursor_created_at')::timestamptz, sqlc.narg('cursor_id')::text))
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('page_size');

-- name: InsertOrder :exec
INSERT INTO orders (id, customer_id, currency, total_minor, status, aggregate_version, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $7);

-- name: InsertOrderItem :exec
INSERT INTO order_items (order_id, line_number, product_id, quantity, unit_price_minor, line_total_minor, quote_id, quote_version)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: UpdateOrderCancellation :one
UPDATE orders SET status = 'CANCEL_REQUESTED', aggregate_version = aggregate_version + 1, updated_at = $3
WHERE id = $1 AND aggregate_version = $2 AND status IN ('PENDING', 'CONFIRMED')
RETURNING id, customer_id, currency, total_minor, status, failure_code, reservation_id,
          payment_id, shipment_id, inventory_released, payment_refunded,
          aggregate_version, created_at, updated_at;

-- name: UpdateOrderWorkflowState :execrows
UPDATE orders SET status = $2, failure_code = COALESCE($3, failure_code), reservation_id = COALESCE($4, reservation_id), payment_id = COALESCE($5, payment_id),
       shipment_id = COALESCE($6, shipment_id), total_minor = COALESCE($7, total_minor), aggregate_version = aggregate_version + 1, updated_at = $8
WHERE id = $1 AND aggregate_version = $9;

-- name: MarkInventoryReleased :execrows
UPDATE orders
SET inventory_released = true, aggregate_version = aggregate_version + 1, updated_at = $2
WHERE id = $1 AND aggregate_version = $3 AND NOT inventory_released;

-- name: MarkPaymentRefunded :execrows
UPDATE orders
SET payment_refunded = true, aggregate_version = aggregate_version + 1, updated_at = $2
WHERE id = $1 AND aggregate_version = $3 AND NOT payment_refunded;

-- name: UpdateOrderItemQuote :exec
UPDATE order_items SET unit_price_minor = $3, line_total_minor = $4, quote_id = $5, quote_version = $6
WHERE order_id = $1 AND product_id = $2;

-- name: InsertSaga :exec
INSERT INTO saga_instances (id, order_id, phase, version, created_at, updated_at)
VALUES ($1, $2, $3, 1, $4, $4);

-- name: GetSagaByOrderID :one
SELECT id, order_id, phase, version, failure_class, created_at, updated_at
FROM saga_instances WHERE order_id = $1;

-- name: UpdateSagaPhase :exec
UPDATE saga_instances SET phase = $2, version = version + 1, updated_at = $3
WHERE id = $1;

-- name: InsertIdempotency :execrows
INSERT INTO idempotency_records (actor_id, operation, key, request_hash, status, resource_id, response_status, response_body, created_at, expires_at)
VALUES ($1, $2, $3, $4, 'COMPLETED', $5, $6, $7, $8, $9)
ON CONFLICT (actor_id, operation, key) DO NOTHING;

-- name: GetIdempotency :one
SELECT actor_id, operation, key, request_hash, status, resource_id, response_status, response_body,
       created_at, expires_at
FROM idempotency_records WHERE actor_id = $1 AND operation = $2 AND key = $3;

-- name: UpdateIdempotencyResponse :exec
UPDATE idempotency_records SET resource_id = $4, response_body = $5
WHERE actor_id = $1 AND operation = $2 AND key = $3;

-- name: InsertOutbox :exec
INSERT INTO outbox_events (id, topic, partition_key, message_type, aggregate_type, aggregate_id, payload, headers, created_at, next_attempt_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9);

-- name: ListPendingOutbox :many
SELECT id, topic, partition_key, message_type, aggregate_type, aggregate_id, payload, headers,
       created_at, published_at, attempts, next_attempt_at, lease_owner, lease_until, last_error
FROM outbox_events
WHERE published_at IS NULL AND next_attempt_at <= now()
ORDER BY created_at
LIMIT $1;

-- name: ClaimPendingOutbox :many
UPDATE outbox_events
SET lease_owner = $1, lease_until = $2
WHERE id IN (
  SELECT id FROM outbox_events
  WHERE published_at IS NULL
    AND next_attempt_at <= now()
    AND (lease_until IS NULL OR lease_until < now())
  ORDER BY created_at
  FOR UPDATE SKIP LOCKED
  LIMIT $3
)
RETURNING id, topic, partition_key, message_type, aggregate_type, aggregate_id, payload, headers,
          created_at, published_at, attempts, next_attempt_at, lease_owner, lease_until, last_error;

-- name: MarkOutboxPublished :exec
UPDATE outbox_events SET published_at = $2, attempts = attempts + 1, lease_owner = NULL, lease_until = NULL, last_error = NULL
WHERE id = $1 AND published_at IS NULL;

-- name: MarkOutboxFailed :exec
UPDATE outbox_events SET attempts = attempts + 1, last_error = $2, next_attempt_at = $3, lease_owner = NULL, lease_until = NULL
WHERE id = $1 AND published_at IS NULL;

-- name: InsertInbox :execrows
INSERT INTO inbox_messages (consumer_group, message_id, message_type, payload_hash, received_at, status, attempts, correlation_id, causation_id)
VALUES ($1, $2, $3, $4, $5, 'RECEIVED', 1, $6, $7)
ON CONFLICT (consumer_group, message_id) DO NOTHING;

-- name: GetInbox :one
SELECT consumer_group, message_id, message_type, payload_hash, received_at, processed_at,
       status, attempts, next_attempt_at, last_error, correlation_id, causation_id
FROM inbox_messages WHERE consumer_group = $1 AND message_id = $2;

-- name: RecordInboxFailure :one
INSERT INTO inbox_messages (consumer_group, message_id, message_type, payload_hash, received_at, status, attempts, correlation_id, causation_id, last_error, next_attempt_at)
VALUES ($1, $2, $3, $4, $5, 'RETRY', 1, $6, $7, $8, $9)
ON CONFLICT (consumer_group, message_id) DO UPDATE SET
  status = 'RETRY',
  attempts = inbox_messages.attempts + 1,
  last_error = EXCLUDED.last_error,
  next_attempt_at = EXCLUDED.next_attempt_at
RETURNING attempts, status;

-- name: MarkInboxDeadLetter :exec
UPDATE inbox_messages SET status = 'DLQ', last_error = $3, next_attempt_at = NULL
WHERE consumer_group = $1 AND message_id = $2;

-- name: MarkInboxProcessed :exec
UPDATE inbox_messages SET status = 'PROCESSED', processed_at = $3, last_error = NULL
WHERE consumer_group = $1 AND message_id = $2;

-- name: CountPendingOutbox :one
SELECT count(*) FROM outbox_events WHERE published_at IS NULL;

-- name: CountPendingInbox :one
SELECT count(*) FROM inbox_messages WHERE status IN ('RECEIVED', 'RETRY');

-- name: AcquireMigrationLock :exec
SELECT pg_advisory_lock($1::bigint);

-- name: ReleaseMigrationLock :exec
SELECT pg_advisory_unlock($1::bigint);
