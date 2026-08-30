-- +goose Up
CREATE INDEX IF NOT EXISTS users_created_id_desc_idx ON users (created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS users_status_created_id_desc_idx ON users (status, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS users_email_normalized_prefix_idx ON users (email_normalized text_pattern_ops);
CREATE INDEX IF NOT EXISTS user_roles_role_user_idx ON user_roles (role_id, user_id);
CREATE INDEX IF NOT EXISTS sessions_user_created_id_desc_idx ON sessions (user_id, created_at DESC, id DESC);

-- +goose Down
DROP INDEX IF EXISTS sessions_user_created_id_desc_idx;
DROP INDEX IF EXISTS user_roles_role_user_idx;
DROP INDEX IF EXISTS users_email_normalized_prefix_idx;
DROP INDEX IF EXISTS users_status_created_id_desc_idx;
DROP INDEX IF EXISTS users_created_id_desc_idx;
