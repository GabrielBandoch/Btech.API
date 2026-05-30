-- +goose Up
-- +goose StatementBegin

-- 1. Create driver_documents table
CREATE TABLE IF NOT EXISTS driver_documents (
    id VARCHAR(255) PRIMARY KEY,
    organization_id VARCHAR(255) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    driver_id VARCHAR(255) NOT NULL REFERENCES drivers(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL CHECK (type IN ('cnh', 'mopp', 'aso', 'toxicology', 'certification', 'custom')),
    document_number VARCHAR(100) NOT NULL,
    issuing_authority VARCHAR(255),
    issue_date TIMESTAMP WITH TIME ZONE,
    expiration_date TIMESTAMP WITH TIME ZONE,
    notes TEXT,
    status VARCHAR(50) NOT NULL CHECK (status IN ('valid', 'expiring_soon', 'expired')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_driver_documents_org_deleted ON driver_documents(organization_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_driver_documents_driver ON driver_documents(driver_id);
CREATE INDEX IF NOT EXISTS idx_driver_documents_expiry ON driver_documents(organization_id, status, expiration_date);

-- 2. Create driver_document_files table (1:N)
CREATE TABLE IF NOT EXISTS driver_document_files (
    id VARCHAR(255) PRIMARY KEY,
    organization_id VARCHAR(255) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    document_id VARCHAR(255) NOT NULL REFERENCES driver_documents(id) ON DELETE CASCADE,
    file_path VARCHAR(500) NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    file_size INTEGER NOT NULL,
    content_type VARCHAR(100) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_driver_document_files_doc ON driver_document_files(document_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_driver_document_files_org_deleted ON driver_document_files(organization_id, deleted_at);

-- 3. Create driver_document_alerts table for alert thresholds
CREATE TABLE IF NOT EXISTS driver_document_alerts (
    id VARCHAR(255) PRIMARY KEY,
    organization_id VARCHAR(255) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    document_id VARCHAR(255) NOT NULL REFERENCES driver_documents(id) ON DELETE CASCADE,
    days_remaining INTEGER NOT NULL,
    status VARCHAR(50) NOT NULL CHECK (status IN ('active', 'resolved')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    resolved_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_driver_document_alerts_doc ON driver_document_alerts(document_id, status);

-- 4. Seed permissions
INSERT INTO permissions (id, name, description) VALUES
    ('perm_driver_documents_create', 'driver_documents:create', 'Allow creating driver documents'),
    ('perm_driver_documents_read', 'driver_documents:read', 'Allow reading driver documents'),
    ('perm_driver_documents_update', 'driver_documents:update', 'Allow updating driver documents'),
    ('perm_driver_documents_delete', 'driver_documents:delete', 'Allow deleting driver documents')
ON CONFLICT (id) DO NOTHING;

-- Map permissions to roles
INSERT INTO role_permissions (role, permission_id) VALUES
    ('owner', 'perm_driver_documents_create'),
    ('owner', 'perm_driver_documents_read'),
    ('owner', 'perm_driver_documents_update'),
    ('owner', 'perm_driver_documents_delete'),
    ('admin', 'perm_driver_documents_create'),
    ('admin', 'perm_driver_documents_read'),
    ('admin', 'perm_driver_documents_update'),
    ('admin', 'perm_driver_documents_delete'),
    ('operator', 'perm_driver_documents_create'),
    ('operator', 'perm_driver_documents_read'),
    ('operator', 'perm_driver_documents_update'),
    ('operator', 'perm_driver_documents_delete'),
    ('viewer', 'perm_driver_documents_read')
ON CONFLICT DO NOTHING;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM role_permissions WHERE permission_id IN (
    'perm_driver_documents_create', 
    'perm_driver_documents_read', 
    'perm_driver_documents_update', 
    'perm_driver_documents_delete'
);
DELETE FROM permissions WHERE id IN (
    'perm_driver_documents_create', 
    'perm_driver_documents_read', 
    'perm_driver_documents_update', 
    'perm_driver_documents_delete'
);
DROP TABLE IF EXISTS driver_document_alerts;
DROP TABLE IF EXISTS driver_document_files;
DROP TABLE IF EXISTS driver_documents;
-- +goose StatementEnd
