-- name: CreatePlan :one
INSERT INTO plans (id, name, slug, description, type, price_monthly, price_yearly, is_trial, trial_days, tier, is_active, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
RETURNING *;

-- name: GetPlanByID :one
SELECT * FROM plans WHERE id = $1;

-- name: GetPlanBySlug :one
SELECT * FROM plans WHERE slug = $1;

-- name: ListPlans :many
SELECT * FROM plans WHERE is_active = TRUE ORDER BY tier ASC;

-- name: ListAllPlans :many
SELECT * FROM plans ORDER BY tier ASC;

-- name: UpdatePlan :exec
UPDATE plans
SET name = $2, slug = $3, description = $4, type = $5, price_monthly = $6, price_yearly = $7,
    is_trial = $8, trial_days = $9, tier = $10, is_active = $11, updated_at = NOW()
WHERE id = $1;

-- name: DeletePlan :exec
UPDATE plans SET is_active = FALSE, updated_at = NOW() WHERE id = $1;

-- name: ExistsPlanBySlug :one
SELECT EXISTS(SELECT 1 FROM plans WHERE slug = $1);

-- name: CreatePlanFeature :one
INSERT INTO plan_features (id, plan_id, feature_key, feature_type, value_bool, value_int, value_text, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetPlanFeatures :many
SELECT * FROM plan_features WHERE plan_id = $1 ORDER BY feature_key ASC;

-- name: GetPlanFeatureByKey :one
SELECT * FROM plan_features WHERE plan_id = $1 AND feature_key = $2;

-- name: UpdatePlanFeature :exec
UPDATE plan_features
SET feature_type = $3, value_bool = $4, value_int = $5, value_text = $6
WHERE plan_id = $1 AND feature_key = $2;

-- name: UpsertPlanFeature :one
INSERT INTO plan_features (id, plan_id, feature_key, feature_type, value_bool, value_int, value_text, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (plan_id, feature_key)
DO UPDATE SET feature_type = EXCLUDED.feature_type, value_bool = EXCLUDED.value_bool, value_int = EXCLUDED.value_int, value_text = EXCLUDED.value_text
RETURNING *;

-- name: DeletePlanFeature :exec
DELETE FROM plan_features WHERE plan_id = $1 AND feature_key = $2;

-- name: DeleteAllPlanFeatures :exec
DELETE FROM plan_features WHERE plan_id = $1;
