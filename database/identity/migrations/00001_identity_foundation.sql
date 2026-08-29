-- +goose Up
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    email TEXT NOT NULL,
    email_normalized TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('ACTIVE', 'DISABLED', 'PENDING_VERIFICATION')),
    email_verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS roles (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'DISABLED')),
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS permissions (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_id UUID NOT NULL REFERENCES roles(id),
    permission_id UUID NOT NULL REFERENCES permissions(id),
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE IF NOT EXISTS user_roles (
    user_id UUID NOT NULL REFERENCES users(id),
    role_id UUID NOT NULL REFERENCES roles(id),
    assigned_at TIMESTAMPTZ NOT NULL,
    assigned_by UUID REFERENCES users(id),
    PRIMARY KEY (user_id, role_id)
);

CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY,
    family_id UUID NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id),
    refresh_token_hash BYTEA NOT NULL UNIQUE,
    status TEXT NOT NULL CHECK (status IN ('ACTIVE', 'ROTATED', 'REVOKED')),
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    revoked_reason TEXT,
    rotated_from_session_id UUID REFERENCES sessions(id),
    replaced_by_session_id UUID REFERENCES sessions(id),
    created_at TIMESTAMPTZ NOT NULL,
    last_used_at TIMESTAMPTZ,
    user_agent_hash BYTEA,
    ip_prefix_hash BYTEA
);

CREATE INDEX IF NOT EXISTS sessions_user_active_idx ON sessions(user_id, status);
CREATE INDEX IF NOT EXISTS sessions_family_idx ON sessions(family_id, status);

CREATE TABLE IF NOT EXISTS reset_tokens (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    token_hash BYTEA NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS verification_tokens (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    token_hash BYTEA NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS service_principals (
    id UUID PRIMARY KEY,
    client_id TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL CHECK (status IN ('ACTIVE', 'DISABLED')),
    scopes JSONB NOT NULL,
    audience TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    last_used_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS service_principal_credentials (
    id UUID PRIMARY KEY,
    service_principal_id UUID NOT NULL REFERENCES service_principals(id),
    secret_hash TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('ACTIVE', 'RETIRED')),
    valid_from TIMESTAMPTZ NOT NULL,
    retired_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL,
    last_used_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS service_principal_credentials_active_idx
    ON service_principal_credentials(service_principal_id, status, valid_from);

CREATE TABLE IF NOT EXISTS security_audit (
    id UUID PRIMARY KEY,
    actor_id TEXT,
    actor_type TEXT,
    action TEXT NOT NULL,
    target_type TEXT,
    target_id TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    correlation_id TEXT NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS outbox_events (
    id UUID PRIMARY KEY,
    event_type TEXT NOT NULL,
    aggregate_type TEXT NOT NULL,
    aggregate_id TEXT NOT NULL,
    payload JSONB NOT NULL,
    headers JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL,
    published_at TIMESTAMPTZ,
    attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    last_error TEXT
);

CREATE INDEX IF NOT EXISTS outbox_unpublished_idx ON outbox_events(created_at) WHERE published_at IS NULL;

-- +goose StatementBegin
INSERT INTO permissions (id, name, description, created_at)
VALUES
    ('00000000-0000-0000-0000-000000000001', 'orders:read:self', 'Read owned orders', now()),
    ('00000000-0000-0000-0000-000000000002', 'orders:read:any', 'Read any order', now()),
    ('00000000-0000-0000-0000-000000000003', 'orders:cancel', 'Cancel permitted orders', now()),
    ('00000000-0000-0000-0000-000000000004', 'payments:refund', 'Refund payments', now()),
    ('00000000-0000-0000-0000-000000000005', 'identity:users:read', 'Read identity users', now()),
    ('00000000-0000-0000-0000-000000000006', 'identity:roles:manage', 'Manage identity roles', now())
ON CONFLICT (name) DO NOTHING;
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO roles (id, name, description, status, created_at, updated_at)
VALUES
    ('00000000-0000-0000-0000-000000000011', 'customer', 'Customer access', 'ACTIVE', now(), now()),
    ('00000000-0000-0000-0000-000000000012', 'support', 'Support operations', 'ACTIVE', now(), now()),
    ('00000000-0000-0000-0000-000000000013', 'finance', 'Finance operations', 'ACTIVE', now(), now()),
    ('00000000-0000-0000-0000-000000000014', 'admin', 'Platform administration', 'ACTIVE', now(), now())
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT '00000000-0000-0000-0000-000000000011', id FROM permissions WHERE name = 'orders:read:self'
UNION ALL SELECT '00000000-0000-0000-0000-000000000012', id FROM permissions WHERE name IN ('orders:read:any', 'orders:cancel')
UNION ALL SELECT '00000000-0000-0000-0000-000000000013', id FROM permissions WHERE name = 'payments:refund'
UNION ALL SELECT '00000000-0000-0000-0000-000000000014', id FROM permissions WHERE name IN ('identity:users:read', 'identity:roles:manage')
ON CONFLICT DO NOTHING;
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS outbox_events;
DROP TABLE IF EXISTS security_audit;
DROP TABLE IF EXISTS service_principal_credentials;
DROP TABLE IF EXISTS service_principals;
DROP TABLE IF EXISTS verification_tokens;
DROP TABLE IF EXISTS reset_tokens;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS users;
