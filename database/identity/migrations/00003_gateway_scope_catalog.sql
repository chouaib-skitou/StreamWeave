-- +goose Up
-- Gateway-facing scopes are kept in Identity's durable authorization catalog.
-- They are additive so existing installations can migrate without data loss.
INSERT INTO permissions (id, name, description, created_at)
VALUES
    ('00000000-0000-0000-0000-000000000007', 'identity:sessions:read', 'Read own sessions through the public edge', now()),
    ('00000000-0000-0000-0000-000000000008', 'identity:sessions:write', 'Revoke own sessions through the public edge', now()),
    ('00000000-0000-0000-0000-000000000009', 'orders:write:self', 'Create an order for the authenticated customer', now()),
    ('00000000-0000-0000-0000-000000000010', 'orders:write:any', 'Create an order for an authorized operator', now())
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT '00000000-0000-0000-0000-000000000011'::uuid, id FROM permissions
WHERE name IN ('identity:sessions:read', 'identity:sessions:write', 'orders:write:self')
UNION ALL
SELECT '00000000-0000-0000-0000-000000000012'::uuid, id FROM permissions
WHERE name IN ('identity:sessions:read', 'identity:sessions:write', 'orders:write:any')
UNION ALL
SELECT '00000000-0000-0000-0000-000000000013'::uuid, id FROM permissions
WHERE name IN ('identity:sessions:read', 'identity:sessions:write')
UNION ALL
SELECT '00000000-0000-0000-0000-000000000014'::uuid, id FROM permissions
WHERE name IN ('identity:sessions:read', 'identity:sessions:write', 'orders:write:any')
ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions WHERE name IN ('identity:sessions:read', 'identity:sessions:write', 'orders:write:self', 'orders:write:any')
);
DELETE FROM permissions
WHERE name IN ('identity:sessions:read', 'identity:sessions:write', 'orders:write:self', 'orders:write:any');

