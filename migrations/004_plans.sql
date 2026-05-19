-- Plans table (master plan definitions)
CREATE TABLE IF NOT EXISTS plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE,
    slug VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    type VARCHAR(10) NOT NULL DEFAULT 'paid',
    price_monthly NUMERIC(12,2) NOT NULL DEFAULT 0,
    price_yearly NUMERIC(12,2) NOT NULL DEFAULT 0,
    is_trial BOOLEAN NOT NULL DEFAULT FALSE,
    trial_days INT NOT NULL DEFAULT 0,
    tier INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT check_plan_type CHECK (type IN ('free', 'paid')),
    CONSTRAINT check_trial_days CHECK (
        (is_trial = FALSE) OR (is_trial = TRUE AND trial_days > 0)
    )
);

CREATE INDEX IF NOT EXISTS idx_plans_slug ON plans(slug);
CREATE INDEX IF NOT EXISTS idx_plans_type ON plans(type);
CREATE INDEX IF NOT EXISTS idx_plans_is_active ON plans(is_active);
CREATE INDEX IF NOT EXISTS idx_plans_tier ON plans(tier);

-- Plan features table (feature permissions per plan)
CREATE TABLE IF NOT EXISTS plan_features (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_id UUID NOT NULL REFERENCES plans(id) ON DELETE CASCADE,
    feature_key VARCHAR(100) NOT NULL,
    feature_type VARCHAR(20) NOT NULL,
    value_bool BOOLEAN,
    value_int INT,
    value_text VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT check_feature_type CHECK (feature_type IN ('boolean', 'quota', 'tier')),
    CONSTRAINT uq_plan_feature UNIQUE (plan_id, feature_key)
);

CREATE INDEX IF NOT EXISTS idx_plan_features_plan_id ON plan_features(plan_id);
CREATE INDEX IF NOT EXISTS idx_plan_features_key ON plan_features(feature_key);
