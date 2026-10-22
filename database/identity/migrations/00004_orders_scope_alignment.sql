-- +goose Up
-- Replace the legacy broad cancellation scope with explicit ownership scopes.
INSERT INTO permissions (id, name, description, created_at)
VALUES
    ('00000000-0000-0000-0000-000000000015', 'orders:cancel:self', 'Cancel an owned order', now()),
    ('00000000-0000-0000-0000-000000000016', 'orders:cancel:any', 'Cancel an order for an authorized operator', now())
ON CONFLICT (name) DO NOTHING;

DELETE FROM role_permissions
WHERE permission_id = (SELECT id FROM permissions WHERE name = 'orders:cancel');

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.name = 'orders:cancel:self'
WHERE r.name = 'customer' ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.name = 'orders:cancel:any'
WHERE r.name = 'support' ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM role_permissions
WHERE permission_id IN (SELECT id FROM permissions WHERE name IN ('orders:cancel:self', 'orders:cancel:any'));
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.name = 'orders:cancel'
WHERE r.name = 'support' ON CONFLICT DO NOTHING;
DELETE FROM permissions WHERE name IN ('orders:cancel:self', 'orders:cancel:any');
