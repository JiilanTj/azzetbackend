-- name: CreateAdmin :one
INSERT INTO platform_admins (id, email, password_hash, name, role, status, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetAdminByID :one
SELECT * FROM platform_admins WHERE id = $1;

-- name: GetAdminByEmail :one
SELECT * FROM platform_admins WHERE email = $1;

-- name: ListAdmins :many
SELECT * FROM platform_admins WHERE status != 'DELETED' ORDER BY created_at DESC;

-- name: UpdateAdmin :exec
UPDATE platform_admins
SET name = $2, role = $3, status = $4, updated_at = NOW()
WHERE id = $1;

-- name: UpdateAdminPassword :exec
UPDATE platform_admins SET password_hash = $2, updated_at = NOW() WHERE id = $1;

-- name: UpdateAdminMFA :exec
UPDATE platform_admins SET mfa_secret = $2, mfa_enabled = $3, updated_at = NOW() WHERE id = $1;

-- name: UpdateAdminLastLogin :exec
UPDATE platform_admins SET last_login_at = NOW(), last_login_ip = $2, updated_at = NOW() WHERE id = $1;

-- name: DeleteAdmin :exec
UPDATE platform_admins SET status = 'DELETED', updated_at = NOW() WHERE id = $1;

-- name: ExistsAdminByEmail :one
SELECT EXISTS(SELECT 1 FROM platform_admins WHERE email = $1);
