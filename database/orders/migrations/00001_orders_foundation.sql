-- +goose Up
CREATE TABLE orders (
    id text PRIMARY KEY CHECK (id ~ '^ord_[A-Za-z0-9]+$'),
    customer_id text NOT NULL CHECK (customer_id ~ '^usr_[A-Za-z0-9]+$'),
    currency char(3) NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    total_minor bigint CHECK (total_minor IS NULL OR (total_minor >= 0 AND total_minor <= 1000000000)),
    status text NOT NULL CHECK (status IN ('PENDING', 'CONFIRMED', 'CANCEL_REQUESTED', 'CANCELLED', 'COMPLETED', 'FAILED')),
    failure_code text,
    reservation_id text,
    payment_id text,
    shipment_id text,
    aggregate_version bigint NOT NULL CHECK (aggregate_version > 0),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);

CREATE INDEX orders_customer_created_idx ON orders (customer_id, created_at DESC, id DESC);

CREATE TABLE order_items (
    order_id text NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    line_number integer NOT NULL CHECK (line_number > 0),
    product_id text NOT NULL CHECK (product_id ~ '^prd_[A-Za-z0-9]+$'),
    quantity integer NOT NULL CHECK (quantity BETWEEN 1 AND 1000),
    unit_price_minor bigint CHECK (unit_price_minor IS NULL OR unit_price_minor BETWEEN 0 AND 1000000000),
    line_total_minor bigint CHECK (line_total_minor IS NULL OR line_total_minor BETWEEN 0 AND 1000000000),
    quote_id text,
    quote_version bigint,
    PRIMARY KEY (order_id, line_number),
    UNIQUE (order_id, product_id)
);

CREATE TABLE idempotency_records (
    actor_id text NOT NULL,
    operation text NOT NULL,
    key text NOT NULL,
    request_hash text NOT NULL,
    status text NOT NULL,
    resource_id text,
    response_status integer,
    response_body jsonb,
    created_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    PRIMARY KEY (actor_id, operation, key)
);

-- +goose Down
DROP TABLE IF EXISTS idempotency_records;
DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;
