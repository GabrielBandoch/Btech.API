-- +goose Up
-- +goose StatementBegin

-- 1. Create fuel_records table
CREATE TABLE IF NOT EXISTS fuel_records (
    id                  VARCHAR(255) PRIMARY KEY,
    organization_id     VARCHAR(255) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    vehicle_id          VARCHAR(255) NOT NULL REFERENCES vehicles(id) ON DELETE CASCADE,
    driver_id           VARCHAR(255) REFERENCES drivers(id) ON DELETE SET NULL,
    date                TIMESTAMP WITH TIME ZONE NOT NULL,
    liters              NUMERIC(8, 3) NOT NULL CHECK (liters > 0),
    price_per_liter     NUMERIC(10, 4) NOT NULL CHECK (price_per_liter > 0),
    total_cost          NUMERIC(15, 2) NOT NULL CHECK (total_cost > 0),
    odometer_reading    INTEGER NOT NULL CHECK (odometer_reading >= 0),
    fuel_type           VARCHAR(50) NOT NULL CHECK (fuel_type IN ('gasoline', 'ethanol', 'diesel', 'gnv', 'electric')),
    station_name        VARCHAR(255),
    notes               TEXT,
    is_anomaly          BOOLEAN NOT NULL DEFAULT FALSE,
    created_at          TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at          TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    deleted_at          TIMESTAMP WITH TIME ZONE
);

-- Primary tenant + soft-delete query pattern
CREATE INDEX IF NOT EXISTS idx_fuel_records_org_deleted
    ON fuel_records(organization_id, deleted_at);

-- Vehicle-level filtering (most common filter in list and anomaly check)
CREATE INDEX IF NOT EXISTS idx_fuel_records_vehicle
    ON fuel_records(vehicle_id);

-- Date range filtering for reports and dashboard aggregations
CREATE INDEX IF NOT EXISTS idx_fuel_records_org_date
    ON fuel_records(organization_id, date DESC);

-- Partial index for anomaly dashboard queries (only anomalous rows)
CREATE INDEX IF NOT EXISTS idx_fuel_records_org_anomaly
    ON fuel_records(organization_id, is_anomaly) WHERE is_anomaly = TRUE;

-- 2. Seed fuel permissions
INSERT INTO permissions (id, name, description) VALUES
    ('perm_fuel_create', 'fuel:create', 'Allow recording fuel fill-ups'),
    ('perm_fuel_read',   'fuel:read',   'Allow reading fuel records and reports'),
    ('perm_fuel_update', 'fuel:update', 'Allow updating fuel records'),
    ('perm_fuel_delete', 'fuel:delete', 'Allow deleting fuel records')
ON CONFLICT (id) DO NOTHING;

-- Map permissions to roles
-- owner + admin: full access
-- operator: create/read/update — cannot delete (financial data protection)
-- viewer: read-only
INSERT INTO role_permissions (role, permission_id) VALUES
    ('owner',    'perm_fuel_create'),
    ('owner',    'perm_fuel_read'),
    ('owner',    'perm_fuel_update'),
    ('owner',    'perm_fuel_delete'),
    ('admin',    'perm_fuel_create'),
    ('admin',    'perm_fuel_read'),
    ('admin',    'perm_fuel_update'),
    ('admin',    'perm_fuel_delete'),
    ('operator', 'perm_fuel_create'),
    ('operator', 'perm_fuel_read'),
    ('operator', 'perm_fuel_update'),
    ('viewer',   'perm_fuel_read')
ON CONFLICT DO NOTHING;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DELETE FROM role_permissions WHERE permission_id IN (
    'perm_fuel_create', 'perm_fuel_read', 'perm_fuel_update', 'perm_fuel_delete'
);
DELETE FROM permissions WHERE id IN (
    'perm_fuel_create', 'perm_fuel_read', 'perm_fuel_update', 'perm_fuel_delete'
);
DROP TABLE IF EXISTS fuel_records;

-- +goose StatementEnd
