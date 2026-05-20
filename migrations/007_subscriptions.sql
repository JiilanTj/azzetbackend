-- Tenant Subscriptions: Links workspace (entity) to a plan
CREATE TABLE IF NOT EXISTS tenant_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    plan_id UUID NOT NULL REFERENCES plans(id),
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    billing_cycle VARCHAR(10),
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    trial_ends_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT check_sub_status CHECK (
        status IN ('active', 'trial', 'expired', 'cancelled')
    ),
    CONSTRAINT check_billing_cycle CHECK (
        billing_cycle IS NULL OR billing_cycle IN ('monthly', 'yearly')
    )
);

CREATE INDEX IF NOT EXISTS idx_subscriptions_workspace ON tenant_subscriptions(workspace_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_plan ON tenant_subscriptions(plan_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_status ON tenant_subscriptions(status);
CREATE INDEX IF NOT EXISTS idx_subscriptions_workspace_active ON tenant_subscriptions(workspace_id, status)
    WHERE status IN ('active', 'trial');

-- Tenant Usage: Tracks quota consumption per workspace per period
CREATE TABLE IF NOT EXISTS tenant_usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    feature_key VARCHAR(100) NOT NULL,
    usage_count INT NOT NULL DEFAULT 0,
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_tenant_usage UNIQUE (workspace_id, feature_key, period_start)
);

CREATE INDEX IF NOT EXISTS idx_usage_workspace ON tenant_usage(workspace_id);
CREATE INDEX IF NOT EXISTS idx_usage_workspace_period ON tenant_usage(workspace_id, period_start, period_end);
CREATE INDEX IF NOT EXISTS idx_usage_feature ON tenant_usage(workspace_id, feature_key);
