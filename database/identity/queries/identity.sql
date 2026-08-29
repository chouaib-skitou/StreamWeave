-- name: GetUserByEmail :one
SELECT id, email, email_normalized, password_hash, status, email_verified_at, created_at, updated_at
FROM users WHERE email_normalized = $1;

-- name: GetUserByID :one
SELECT id, email, email_normalized, password_hash, status, email_verified_at, created_at, updated_at
FROM users WHERE id = $1;

-- name: CreateUser :one
INSERT INTO users (id, email, email_normalized, password_hash, status, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $6)
RETURNING id, email, email_normalized, password_hash, status, email_verified_at, created_at, updated_at;

-- name: UpdateUserStatus :exec
UPDATE users SET status = $2, updated_at = $3 WHERE id = $1;

-- name: UpdatePassword :exec
UPDATE users SET password_hash = $2, updated_at = $3 WHERE id = $1;

-- name: MarkEmailVerified :exec
UPDATE users SET email_verified_at = $2, status = 'ACTIVE', updated_at = $2 WHERE id = $1;

-- name: ListUserRoles :many
SELECT r.name AS role_name, p.name AS permission_name
FROM user_roles ur
JOIN roles r ON r.id = ur.role_id
JOIN role_permissions rp ON rp.role_id = r.id
JOIN permissions p ON p.id = rp.permission_id
WHERE ur.user_id = $1 AND r.status = 'ACTIVE'
ORDER BY r.name, p.name;

-- name: AssignRole :exec
INSERT INTO user_roles (user_id, role_id, assigned_at, assigned_by)
SELECT $1, id, $3, $4 FROM roles WHERE name = $2 AND status = 'ACTIVE'
ON CONFLICT (user_id, role_id) DO NOTHING;

-- name: RemoveRole :exec
DELETE FROM user_roles ur USING roles r
WHERE ur.user_id = $1 AND ur.role_id = r.id AND r.name = $2;

-- name: CountActiveAdministrators :one
SELECT count(*) FROM user_roles ur
JOIN roles r ON r.id = ur.role_id
WHERE r.name = 'admin' AND r.status = 'ACTIVE';

-- name: GetRoleByName :one
SELECT id, name, description, status, created_at, updated_at FROM roles WHERE name = $1;

-- name: CreateSession :one
INSERT INTO sessions (id, family_id, user_id, refresh_token_hash, status, expires_at, created_at, last_used_at, user_agent_hash, ip_prefix_hash)
VALUES ($1, $2, $3, $4, 'ACTIVE', $5, $6, $6, $7, $8)
RETURNING id, family_id, user_id, refresh_token_hash, status, expires_at, revoked_at, revoked_reason, rotated_from_session_id, replaced_by_session_id, created_at, last_used_at;

-- name: GetSessionByRefreshHash :one
SELECT id, family_id, user_id, refresh_token_hash, status, expires_at, revoked_at, revoked_reason, rotated_from_session_id, replaced_by_session_id, created_at, last_used_at
FROM sessions WHERE refresh_token_hash = $1;

-- name: RotateSession :execrows
UPDATE sessions SET status = 'ROTATED', replaced_by_session_id = $2, last_used_at = $3 WHERE id = $1 AND status = 'ACTIVE';

-- name: RevokeSessionFamily :exec
UPDATE sessions SET status = 'REVOKED', revoked_at = $2, revoked_reason = $3
WHERE family_id = $1 AND status <> 'REVOKED';

-- name: RevokeSessionByID :exec
UPDATE sessions SET status = 'REVOKED', revoked_at = $2, revoked_reason = $3
WHERE id = $1 AND status <> 'REVOKED';

-- name: RevokeUserSessions :exec
UPDATE sessions SET status = 'REVOKED', revoked_at = $2, revoked_reason = $3
WHERE user_id = $1 AND status <> 'REVOKED';

-- name: ListUserSessions :many
SELECT id, family_id, user_id, status, expires_at, revoked_at, created_at, last_used_at
FROM sessions WHERE user_id = $1 ORDER BY created_at DESC;

-- name: InsertSecurityAudit :exec
INSERT INTO security_audit (id, actor_id, actor_type, action, target_type, target_id, metadata, correlation_id, occurred_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: InsertOutboxEvent :exec
INSERT INTO outbox_events (id, event_type, aggregate_type, aggregate_id, payload, headers, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: ListPendingOutbox :many
SELECT id, event_type, aggregate_type, aggregate_id, payload, headers, created_at, published_at, attempts, last_error
FROM outbox_events WHERE published_at IS NULL ORDER BY created_at LIMIT $1;

-- name: MarkOutboxPublished :exec
UPDATE outbox_events SET published_at = $2, attempts = attempts + 1, last_error = NULL WHERE id = $1;

-- name: MarkOutboxFailed :exec
UPDATE outbox_events SET attempts = attempts + 1, last_error = $2 WHERE id = $1;

-- name: GetServicePrincipal :one
SELECT id, client_id, status, scopes, audience, created_at, updated_at, last_used_at
FROM service_principals WHERE client_id = $1;

-- name: ListActiveServiceCredentials :many
SELECT id, service_principal_id, secret_hash, status, valid_from, retired_at, created_at, last_used_at
FROM service_principal_credentials
WHERE service_principal_id = $1 AND status = 'ACTIVE' AND valid_from <= $2
ORDER BY valid_from DESC;

-- name: MarkServicePrincipalUsed :exec
UPDATE service_principals SET last_used_at = $2 WHERE id = $1;

-- name: CreateResetToken :exec
INSERT INTO reset_tokens (id, user_id, token_hash, expires_at, created_at) VALUES ($1, $2, $3, $4, $5);

-- name: GetResetToken :one
SELECT id, user_id, token_hash, expires_at, consumed_at, created_at FROM reset_tokens WHERE token_hash = $1;

-- name: ConsumeResetToken :execrows
UPDATE reset_tokens SET consumed_at = $2 WHERE id = $1 AND consumed_at IS NULL;

-- name: CreateVerificationToken :exec
INSERT INTO verification_tokens (id, user_id, token_hash, expires_at, created_at) VALUES ($1, $2, $3, $4, $5);

-- name: GetVerificationToken :one
SELECT id, user_id, token_hash, expires_at, consumed_at, created_at FROM verification_tokens WHERE token_hash = $1;

-- name: ConsumeVerificationToken :execrows
UPDATE verification_tokens SET consumed_at = $2 WHERE id = $1 AND consumed_at IS NULL;
