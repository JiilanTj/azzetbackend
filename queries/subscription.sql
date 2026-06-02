-- name: CreateSubscription :one
INSERT INTO tenant_subscriptions (id, workspace_id, plan_id, status, billing_cycle, started_at, expires_at, trial_ends_at, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: GetActiveSubscription :one
SELECT * FROM tenant_subscriptions
WHERE workspace_id = $1 AND status IN ('active', 'trial')
ORDER BY created_at DESC
LIMIT 1;

-- name: GetBlockingSubscription :one
SELECT * FROM tenant_subscriptions
WHERE workspace_id = $1 AND status IN ('active', 'trial', 'pending_payment')
ORDER BY created_at DESC
LIMIT 1;

-- name: GetSubscriptionByID :one
SELECT * FROM tenant_subscriptions WHERE id = $1;

-- name: ListSubscriptionsByWorkspace :many
SELECT * FROM tenant_subscriptions
WHERE workspace_id = $1
ORDER BY created_at DESC;

-- name: UpdateSubscriptionStatus :exec
UPDATE tenant_subscriptions SET status = $2, updated_at = NOW() WHERE id = $1;

-- name: CancelSubscription :exec
UPDATE tenant_subscriptions SET status = 'cancelled', cancelled_at = NOW(), updated_at = NOW() WHERE id = $1;

-- name: ExpireSubscription :exec
UPDATE tenant_subscriptions SET status = 'expired', updated_at = NOW()
WHERE id = $1;

-- name: ExpireTrialSubscriptions :exec
UPDATE tenant_subscriptions SET status = 'expired', updated_at = NOW()
WHERE status = 'trial' AND trial_ends_at < NOW();

-- name: GetSubscriptionWithPlan :one
SELECT ts.*, p.name as plan_name, p.slug as plan_slug, p.type as plan_type, p.tier as plan_tier
FROM tenant_subscriptions ts
JOIN plans p ON ts.plan_id = p.id
WHERE ts.workspace_id = $1 AND ts.status IN ('active', 'trial')
ORDER BY ts.created_at DESC
LIMIT 1;

-- name: ListAllSubscriptions :many
SELECT ts.*, p.name as plan_name, p.slug as plan_slug
FROM tenant_subscriptions ts
JOIN plans p ON ts.plan_id = p.id
ORDER BY ts.created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpsertUsage :one
INSERT INTO tenant_usage (id, workspace_id, feature_key, usage_count, period_start, period_end, created_at, updated_at)
VALUES ($1, $2, $3, 1, $4, $5, $6, $7)
ON CONFLICT (workspace_id, feature_key, period_start)
DO UPDATE SET usage_count = tenant_usage.usage_count + 1, updated_at = NOW()
RETURNING *;

-- name: GetUsage :one
SELECT * FROM tenant_usage
WHERE workspace_id = $1 AND feature_key = $2 AND period_start <= $3 AND period_end > $3;

-- name: GetWorkspaceUsage :many
SELECT * FROM tenant_usage
WHERE workspace_id = $1 AND period_start <= $2 AND period_end > $2
ORDER BY feature_key;

-- name: ResetUsage :exec
DELETE FROM tenant_usage WHERE period_end < NOW();
