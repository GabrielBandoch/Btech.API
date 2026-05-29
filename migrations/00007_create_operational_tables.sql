-- +goose Up
-- +goose StatementBegin

-- 1. Create drivers table
CREATE TABLE IF NOT EXISTS drivers (
    id VARCHAR(255) PRIMARY KEY,
    organization_id VARCHAR(255) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    avatar TEXT,
    status VARCHAR(50) NOT NULL CHECK (status IN ('active', 'inactive', 'blocked')),
    score INTEGER NOT NULL DEFAULT 100,
    trips_count INTEGER NOT NULL DEFAULT 0,
    incidents_count INTEGER NOT NULL DEFAULT 0,
    next_scale VARCHAR(255),
    role VARCHAR(100),
    license_expiry VARCHAR(100),
    toxicology_expiry VARCHAR(100),
    training_expiry VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_drivers_org_deleted ON drivers(organization_id, deleted_at);

-- 2. Create vehicles table
CREATE TABLE IF NOT EXISTS vehicles (
    id VARCHAR(255) PRIMARY KEY,
    organization_id VARCHAR(255) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    placa VARCHAR(50) NOT NULL,
    brand VARCHAR(100) NOT NULL,
    model VARCHAR(100) NOT NULL,
    year INTEGER NOT NULL,
    type VARCHAR(100) NOT NULL,
    mileage INTEGER NOT NULL DEFAULT 0,
    status VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_vehicles_org_deleted ON vehicles(organization_id, deleted_at);

-- 3. Create trips table
CREATE TABLE IF NOT EXISTS trips (
    id VARCHAR(255) PRIMARY KEY,
    organization_id VARCHAR(255) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    origin VARCHAR(255) NOT NULL,
    destination VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL,
    driver_name VARCHAR(255),
    driver_avatar TEXT,
    vehicle_placa VARCHAR(50),
    vehicle_model VARCHAR(100),
    cargo_type VARCHAR(100),
    cargo_value NUMERIC(15,2) NOT NULL DEFAULT 0.00,
    cargo_weight INTEGER NOT NULL DEFAULT 0,
    temperature_required VARCHAR(50),
    estimated_time VARCHAR(100),
    speed INTEGER NOT NULL DEFAULT 0,
    fuel_level INTEGER NOT NULL DEFAULT 100,
    last_signal_time VARCHAR(100),
    current_location VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_trips_org_deleted ON trips(organization_id, deleted_at);

-- 4. Create trip_checkpoints table
CREATE TABLE IF NOT EXISTS trip_checkpoints (
    id VARCHAR(255) PRIMARY KEY,
    trip_id VARCHAR(255) NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    sequence INTEGER NOT NULL,
    name VARCHAR(255) NOT NULL,
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,
    planned_at VARCHAR(100),
    arrived_at VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE INDEX idx_trip_checkpoints_trip ON trip_checkpoints(trip_id);

-- 5. Create incidents table
CREATE TABLE IF NOT EXISTS incidents (
    id VARCHAR(255) PRIMARY KEY,
    organization_id VARCHAR(255) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    trip_id VARCHAR(255) REFERENCES trips(id) ON DELETE SET NULL,
    vehicle_placa VARCHAR(50),
    driver_name VARCHAR(255),
    type VARCHAR(100),
    severity VARCHAR(50) NOT NULL CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    description TEXT,
    timestamp VARCHAR(100),
    location VARCHAR(255),
    status VARCHAR(50),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_incidents_org_deleted ON incidents(organization_id, deleted_at);

-- 6. Seed Vehicle permissions
INSERT INTO permissions (id, name, description) VALUES
    ('perm_vehicles_create', 'vehicles:create', 'Allow creating vehicles'),
    ('perm_vehicles_read', 'vehicles:read', 'Allow reading vehicle details'),
    ('perm_vehicles_update', 'vehicles:update', 'Allow updating vehicles'),
    ('perm_vehicles_delete', 'vehicles:delete', 'Allow deleting vehicles')
ON CONFLICT (id) DO NOTHING;

-- Map permissions to roles
-- owner and admin get all
INSERT INTO role_permissions (role, permission_id) VALUES
    ('owner', 'perm_vehicles_create'),
    ('owner', 'perm_vehicles_read'),
    ('owner', 'perm_vehicles_update'),
    ('owner', 'perm_vehicles_delete'),
    ('admin', 'perm_vehicles_create'),
    ('admin', 'perm_vehicles_read'),
    ('admin', 'perm_vehicles_update'),
    ('admin', 'perm_vehicles_delete'),
    -- operator gets all
    ('operator', 'perm_vehicles_create'),
    ('operator', 'perm_vehicles_read'),
    ('operator', 'perm_vehicles_update'),
    ('operator', 'perm_vehicles_delete'),
    -- viewer gets read only
    ('viewer', 'perm_vehicles_read')
ON CONFLICT DO NOTHING;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM role_permissions WHERE permission_id IN ('perm_vehicles_create', 'perm_vehicles_read', 'perm_vehicles_update', 'perm_vehicles_delete');
DELETE FROM permissions WHERE id IN ('perm_vehicles_create', 'perm_vehicles_read', 'perm_vehicles_update', 'perm_vehicles_delete');
DROP TABLE IF EXISTS incidents;
DROP TABLE IF EXISTS trip_checkpoints;
DROP TABLE IF EXISTS trips;
DROP TABLE IF EXISTS vehicles;
DROP TABLE IF EXISTS drivers;
-- +goose StatementEnd
