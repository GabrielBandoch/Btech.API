-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS plans (
    id VARCHAR(255) PRIMARY KEY,
    code VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    monthly_price NUMERIC(10, 2) NOT NULL,
    yearly_price NUMERIC(10, 2) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS plan_entitlements (
    plan_id VARCHAR(255) NOT NULL REFERENCES plans(id) ON DELETE CASCADE,
    key VARCHAR(255) NOT NULL,
    value VARCHAR(255) NOT NULL,
    PRIMARY KEY (plan_id, key)
);

CREATE TABLE IF NOT EXISTS subscriptions (
    id VARCHAR(255) PRIMARY KEY,
    organization_id VARCHAR(255) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE UNIQUE,
    plan_id VARCHAR(255) NOT NULL REFERENCES plans(id) ON DELETE RESTRICT,
    status VARCHAR(50) NOT NULL,
    starts_at TIMESTAMP WITH TIME ZONE NOT NULL,
    ends_at TIMESTAMP WITH TIME ZONE NOT NULL,
    trial_ends_at TIMESTAMP WITH TIME ZONE,
    canceled_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS organization_entitlement_overrides (
    id VARCHAR(255) PRIMARY KEY,
    organization_id VARCHAR(255) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    key VARCHAR(255) NOT NULL,
    value VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS subscription_event_history (
    id VARCHAR(255) PRIMARY KEY,
    organization_id VARCHAR(255) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    subscription_id VARCHAR(255) NOT NULL REFERENCES subscriptions(id) ON DELETE CASCADE,
    event_type VARCHAR(100) NOT NULL,
    from_status VARCHAR(50),
    to_status VARCHAR(50) NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS usage_counters (
    id VARCHAR(255) PRIMARY KEY,
    organization_id VARCHAR(255) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    metric_key VARCHAR(255) NOT NULL,
    current_value INTEGER NOT NULL DEFAULT 0,
    billing_period VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT uq_usage_counters UNIQUE (organization_id, metric_key, billing_period)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_subscriptions_org_status ON subscriptions (organization_id, status);
CREATE INDEX IF NOT EXISTS idx_plan_entitlements_plan ON plan_entitlements (plan_id);
CREATE INDEX IF NOT EXISTS idx_overrides_org_key ON organization_entitlement_overrides (organization_id, key);
CREATE INDEX IF NOT EXISTS idx_sub_event_history_org ON subscription_event_history (organization_id);
CREATE INDEX IF NOT EXISTS idx_usage_counters_org_period ON usage_counters (organization_id, billing_period);

-- Seed Plans
INSERT INTO plans (id, code, name, description, monthly_price, yearly_price, is_active) VALUES
('plan-free-id', 'free', 'Free Plan', 'Basic features for small transport agents', 0.00, 0.00, true),
('plan-pro-id', 'pro', 'Pro Plan', 'Advanced operations, reports, and team collaboration', 99.00, 990.00, true),
('plan-enterprise-id', 'enterprise', 'Enterprise Plan', 'Unlimited operations, API access, custom integrations, and dedicated support', 999.00, 9990.00, true)
ON CONFLICT (code) DO NOTHING;

-- Seed Plan Entitlements
INSERT INTO plan_entitlements (plan_id, key, value) VALUES
-- Free Plan
('plan-free-id', 'drivers.max', '5'),
('plan-free-id', 'trips.monthly', '50'),
('plan-free-id', 'users.max', '2'),
('plan-free-id', 'feature.audit_logs', 'false'),
('plan-free-id', 'feature.api_access', 'false'),
('plan-free-id', 'feature.advanced_reports', 'false'),
-- Pro Plan
('plan-pro-id', 'drivers.max', '50'),
('plan-pro-id', 'trips.monthly', '500'),
('plan-pro-id', 'users.max', '10'),
('plan-pro-id', 'feature.audit_logs', 'true'),
('plan-pro-id', 'feature.api_access', 'false'),
('plan-pro-id', 'feature.advanced_reports', 'true'),
-- Enterprise Plan
('plan-enterprise-id', 'drivers.max', 'unlimited'),
('plan-enterprise-id', 'trips.monthly', 'unlimited'),
('plan-enterprise-id', 'users.max', 'unlimited'),
('plan-enterprise-id', 'feature.audit_logs', 'true'),
('plan-enterprise-id', 'feature.api_access', 'true'),
('plan-enterprise-id', 'feature.advanced_reports', 'true')
ON CONFLICT (plan_id, key) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS usage_counters;
DROP TABLE IF EXISTS subscription_event_history;
DROP TABLE IF EXISTS organization_entitlement_overrides;
DROP TABLE IF EXISTS subscriptions;
DROP TABLE IF EXISTS plan_entitlements;
DROP TABLE IF EXISTS plans;
-- +goose StatementEnd
