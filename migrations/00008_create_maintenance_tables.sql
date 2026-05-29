-- +goose Up
-- +goose StatementBegin

-- 1. Create maintenance_suppliers table
CREATE TABLE IF NOT EXISTS maintenance_suppliers (
    id VARCHAR(255) PRIMARY KEY,
    organization_id VARCHAR(255) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    phone VARCHAR(50),
    email VARCHAR(255),
    address VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_maintenance_suppliers_org_deleted ON maintenance_suppliers(organization_id, deleted_at);

-- 2. Create maintenance_plans table
CREATE TABLE IF NOT EXISTS maintenance_plans (
    id VARCHAR(255) PRIMARY KEY,
    organization_id VARCHAR(255) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    vehicle_id VARCHAR(255) NOT NULL REFERENCES vehicles(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    interval_km INTEGER,
    interval_months INTEGER,
    last_maintenance_km INTEGER,
    last_maintenance_date TIMESTAMP WITH TIME ZONE,
    next_due_km INTEGER,
    next_due_date TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_maintenance_plans_org_deleted ON maintenance_plans(organization_id, deleted_at);
CREATE INDEX idx_maintenance_plans_vehicle ON maintenance_plans(vehicle_id);

-- 3. Create maintenances table
CREATE TABLE IF NOT EXISTS maintenances (
    id VARCHAR(255) PRIMARY KEY,
    organization_id VARCHAR(255) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    vehicle_id VARCHAR(255) NOT NULL REFERENCES vehicles(id) ON DELETE CASCADE,
    maintenance_plan_id VARCHAR(255) REFERENCES maintenance_plans(id) ON DELETE SET NULL,
    supplier_id VARCHAR(255) REFERENCES maintenance_suppliers(id) ON DELETE SET NULL,
    type VARCHAR(50) NOT NULL CHECK (type IN ('preventive', 'corrective')),
    priority VARCHAR(50) NOT NULL CHECK (priority IN ('low', 'medium', 'high', 'critical')),
    status VARCHAR(50) NOT NULL CHECK (status IN ('scheduled', 'in_progress', 'completed', 'canceled')),
    date TIMESTAMP WITH TIME ZONE NOT NULL,
    odometer_at_service INTEGER NOT NULL DEFAULT 0,
    downtime_hours NUMERIC(6,2) NOT NULL DEFAULT 0.00,
    cost NUMERIC(15,2) NOT NULL DEFAULT 0.00,
    description TEXT,
    attachments TEXT[] DEFAULT '{}'::TEXT[],
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_maintenances_org_deleted ON maintenances(organization_id, deleted_at);
CREATE INDEX idx_maintenances_vehicle ON maintenances(vehicle_id);

-- 4. Create maintenance_alerts table
CREATE TABLE IF NOT EXISTS maintenance_alerts (
    id VARCHAR(255) PRIMARY KEY,
    organization_id VARCHAR(255) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    vehicle_id VARCHAR(255) NOT NULL REFERENCES vehicles(id) ON DELETE CASCADE,
    maintenance_plan_id VARCHAR(255) REFERENCES maintenance_plans(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL CHECK (type IN ('mileage_due', 'date_due')),
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    status VARCHAR(50) NOT NULL CHECK (status IN ('active', 'resolved')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_maintenance_alerts_org_deleted ON maintenance_alerts(organization_id, deleted_at);
CREATE INDEX idx_maintenance_alerts_vehicle ON maintenance_alerts(vehicle_id);

-- 5. Seed Maintenance permissions
INSERT INTO permissions (id, name, description) VALUES
    ('perm_maintenance_create', 'maintenance:create', 'Allow creating maintenance records and configurations'),
    ('perm_maintenance_read', 'maintenance:read', 'Allow reading maintenance records and configurations'),
    ('perm_maintenance_update', 'maintenance:update', 'Allow updating maintenance records and configurations'),
    ('perm_maintenance_delete', 'maintenance:delete', 'Allow deleting maintenance records and configurations')
ON CONFLICT (id) DO NOTHING;

-- Map permissions to roles
INSERT INTO role_permissions (role, permission_id) VALUES
    ('owner', 'perm_maintenance_create'),
    ('owner', 'perm_maintenance_read'),
    ('owner', 'perm_maintenance_update'),
    ('owner', 'perm_maintenance_delete'),
    ('admin', 'perm_maintenance_create'),
    ('admin', 'perm_maintenance_read'),
    ('admin', 'perm_maintenance_update'),
    ('admin', 'perm_maintenance_delete'),
    ('operator', 'perm_maintenance_create'),
    ('operator', 'perm_maintenance_read'),
    ('operator', 'perm_maintenance_update'),
    ('operator', 'perm_maintenance_delete'),
    ('viewer', 'perm_maintenance_read')
ON CONFLICT DO NOTHING;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM role_permissions WHERE permission_id IN ('perm_maintenance_create', 'perm_maintenance_read', 'perm_maintenance_update', 'perm_maintenance_delete');
DELETE FROM permissions WHERE id IN ('perm_maintenance_create', 'perm_maintenance_read', 'perm_maintenance_update', 'perm_maintenance_delete');
DROP TABLE IF EXISTS maintenance_alerts;
DROP TABLE IF EXISTS maintenances;
DROP TABLE IF EXISTS maintenance_plans;
DROP TABLE IF EXISTS maintenance_suppliers;
-- +goose StatementEnd
