-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS permissions (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS role_permissions (
    role VARCHAR(50) NOT NULL,
    permission_id VARCHAR(255) NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    PRIMARY KEY (role, permission_id)
);

-- Seed default permissions
INSERT INTO permissions (id, name, description) VALUES
    ('perm_drivers_create', 'drivers:create', 'Allow creating drivers'),
    ('perm_drivers_read', 'drivers:read', 'Allow reading driver details'),
    ('perm_drivers_delete', 'drivers:delete', 'Allow deleting drivers'),
    ('perm_trips_create', 'trips:create', 'Allow creating trips'),
    ('perm_trips_read', 'trips:read', 'Allow reading trip details'),
    ('perm_trips_update', 'trips:update', 'Allow updating trips'),
    ('perm_trips_delete', 'trips:delete', 'Allow deleting trips'),
    ('perm_incidents_create', 'incidents:create', 'Allow creating incidents'),
    ('perm_incidents_read', 'incidents:read', 'Allow reading incident details'),
    ('perm_settings_manage', 'settings:manage', 'Allow managing organization settings')
ON CONFLICT (id) DO NOTHING;

-- Seed default role assignments
-- 1. OWNER permissions (All permissions)
INSERT INTO role_permissions (role, permission_id)
SELECT 'owner', id FROM permissions
ON CONFLICT DO NOTHING;

-- 2. ADMIN permissions (All permissions)
INSERT INTO role_permissions (role, permission_id)
SELECT 'admin', id FROM permissions
ON CONFLICT DO NOTHING;

-- 3. OPERATOR permissions
INSERT INTO role_permissions (role, permission_id) VALUES
    ('operator', 'perm_drivers_read'),
    ('operator', 'perm_trips_create'),
    ('operator', 'perm_trips_read'),
    ('operator', 'perm_trips_update'),
    ('operator', 'perm_incidents_create'),
    ('operator', 'perm_incidents_read')
ON CONFLICT DO NOTHING;

-- 4. VIEWER permissions
INSERT INTO role_permissions (role, permission_id) VALUES
    ('viewer', 'perm_drivers_read'),
    ('viewer', 'perm_trips_read'),
    ('viewer', 'perm_incidents_read')
ON CONFLICT DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS permissions;
-- +goose StatementEnd
