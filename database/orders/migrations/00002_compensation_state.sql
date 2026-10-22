-- +goose Up
-- These flags make cancellation compensation durable and idempotent. They are
-- deliberately part of the order aggregate, not Redis or an in-memory relay.
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS inventory_released boolean NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS payment_refunded boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE orders
    DROP COLUMN IF EXISTS payment_refunded,
    DROP COLUMN IF EXISTS inventory_released;
