-- +goose Up
CREATE TABLE orders (
    id text PRIMARY KEY CHECK (id ~ '^ord_[A-Za-z0-9]+$'),
    customer_id text NOT NULL CHECK (customer_id ~ '^usr_[A-Za-z0-9]+$'),
    currency char(3) NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    total_minor bigint CHECK (total_minor IS NULL OR (total_minor >= 0 AND total_minor <= 1000000000)),
    status text NOT NULL CHECK (status IN ('PENDING','CONFIRMED','CANCEL_REQUESTED','CANCELLED','COMPLETED','FAILED')),
    failure_code text CHECK (failure_code IS NULL OR failure_code ~ '^[A-Z][A-Z0-9_]{1,63}$'),
    reservation_id text,
    payment_id text,
    shipment_id text,
    aggregate_version bigint NOT NULL DEFAULT 1 CHECK (aggregate_version > 0),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);
CREATE INDEX orders_customer_created_idx ON orders (customer_id, created_at DESC, id DESC);
CREATE INDEX orders_status_idx ON orders (status, updated_at DESC);

CREATE TABLE order_items (
    order_id text NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    line_number integer NOT NULL CHECK (line_number > 0 AND line_number <= 100),
    product_id text NOT NULL CHECK (product_id ~ '^prd_[A-Za-z0-9]+$'),
    quantity integer NOT NULL CHECK (quantity BETWEEN 1 AND 1000),
    unit_price_minor bigint CHECK (unit_price_minor IS NULL OR (unit_price_minor >= 0 AND unit_price_minor <= 1000000000)),
    line_total_minor bigint CHECK (line_total_minor IS NULL OR (line_total_minor >= 0 AND line_total_minor <= 1000000000)),
    quote_id text CHECK (quote_id IS NULL OR quote_id ~ '^qte_[A-Za-z0-9]+$'),
    quote_version bigint CHECK (quote_version IS NULL OR quote_version > 0),
    PRIMARY KEY (order_id, line_number),
    UNIQUE (order_id, product_id)
);

CREATE TABLE saga_instances (
    id text PRIMARY KEY,
    order_id text NOT NULL UNIQUE REFERENCES orders(id) ON DELETE CASCADE,
    phase text NOT NULL,
    version bigint NOT NULL DEFAULT 1,
    failure_class text,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);

CREATE TABLE saga_steps (
    saga_id text NOT NULL REFERENCES saga_instances(id) ON DELETE CASCADE,
    operation_id text NOT NULL,
    target_service text NOT NULL,
    command_type text NOT NULL,
    status text NOT NULL,
    attempts integer NOT NULL DEFAULT 0,
    next_attempt_at timestamptz,
    completed_at timestamptz,
    last_error text,
    PRIMARY KEY (saga_id, operation_id)
);

CREATE TABLE inbox_messages (
    consumer_group text NOT NULL,
    message_id text NOT NULL,
    message_type text NOT NULL,
    payload_hash text NOT NULL,
    received_at timestamptz NOT NULL,
    processed_at timestamptz,
    status text NOT NULL CHECK (status IN ('RECEIVED','PROCESSED','RETRY','DLQ')),
    attempts integer NOT NULL DEFAULT 0,
    next_attempt_at timestamptz,
    last_error text,
    correlation_id text NOT NULL,
    causation_id text,
    PRIMARY KEY (consumer_group, message_id)
);

CREATE TABLE outbox_events (
    id uuid PRIMARY KEY,
    topic text NOT NULL,
    partition_key text NOT NULL,
    message_type text NOT NULL,
    aggregate_type text NOT NULL,
    aggregate_id text NOT NULL,
    payload jsonb NOT NULL,
    headers jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL,
    published_at timestamptz,
    attempts integer NOT NULL DEFAULT 0,
    next_attempt_at timestamptz NOT NULL,
    lease_owner text,
    lease_until timestamptz,
    last_error text
);
CREATE INDEX outbox_pending_idx ON outbox_events (published_at, next_attempt_at, created_at);

CREATE TABLE orders_audit (
    id uuid PRIMARY KEY,
    actor_id text,
    actor_type text,
    action text NOT NULL,
    target_type text,
    target_id text,
    reason_code text,
    correlation_id text NOT NULL,
    occurred_at timestamptz NOT NULL
);

CREATE TABLE idempotency_records (
    actor_id text NOT NULL,
    operation text NOT NULL,
    key text NOT NULL,
    request_hash text NOT NULL,
    status text NOT NULL CHECK (status IN ('IN_PROGRESS','COMPLETED','FAILED')),
    resource_id text,
    response_status integer,
    response_body jsonb,
    created_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    PRIMARY KEY (actor_id, operation, key)
);
CREATE INDEX idempotency_expiry_idx ON idempotency_records (expires_at);

-- +goose Down
DROP TABLE IF EXISTS idempotency_records;
DROP TABLE IF EXISTS orders_audit;
DROP TABLE IF EXISTS outbox_events;
DROP TABLE IF EXISTS inbox_messages;
DROP TABLE IF EXISTS saga_steps;
DROP TABLE IF EXISTS saga_instances;
DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;
