-- name: CreateUser :one
INSERT INTO users (id, email, whatsapp, password_hash, name, status, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserByWhatsApp :one
SELECT * FROM users WHERE whatsapp = $1;

-- name: ExistsByEmail :one
SELECT EXISTS(SELECT 1 FROM users WHERE email = $1);

-- name: ExistsByWhatsApp :one
SELECT EXISTS(SELECT 1 FROM users WHERE whatsapp = $1);

-- name: UpdateUserStatus :exec
UPDATE users SET status = $2, updated_at = NOW() WHERE id = $1;

-- name: UpdateUserPassword :exec
UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1;

-- name: UpdateUserLastLogin :exec
UPDATE users SET last_login_at = NOW(), last_login_ip = $2, updated_at = NOW() WHERE id = $1;

-- name: VerifyUserEmail :exec
UPDATE users SET email_verified = TRUE, status = 'ACTIVE', updated_at = NOW() WHERE email = $1;

-- name: VerifyUserWhatsApp :exec
UPDATE users SET whatsapp_verified = TRUE, status = 'ACTIVE', updated_at = NOW() WHERE whatsapp = $1;

-- name: ResetPasswordByIdentifier :exec
UPDATE users SET password_hash = $2, updated_at = NOW()
WHERE email = $1 OR whatsapp = $1;

-- name: CreateSession :one
INSERT INTO sessions (id, user_id, refresh_token, device_name, device_type, ip_address, user_agent, expires_at, last_used_at, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: GetSessionByRefreshToken :one
SELECT * FROM sessions WHERE refresh_token = $1;

-- name: GetSessionByID :one
SELECT * FROM sessions WHERE id = $1;

-- name: GetUserSessions :many
SELECT * FROM sessions
WHERE user_id = $1 AND expires_at > NOW()
ORDER BY last_used_at DESC;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE id = $1 AND user_id = $2;

-- name: DeleteSessionByRefreshToken :exec
DELETE FROM sessions WHERE refresh_token = $1;

-- name: DeleteSessionByID :exec
DELETE FROM sessions WHERE id = $1;

-- name: DeleteUserSessions :exec
DELETE FROM sessions WHERE user_id = $1;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions WHERE expires_at < NOW();

-- name: CreateOTP :exec
INSERT INTO otp_codes (id, identifier, identifier_type, code, purpose, attempts, max_attempts, expires_at, created_at)
VALUES ($1, $2, $3, $4, $5, 0, $6, $7, $8);

-- name: GetValidOTP :one
SELECT * FROM otp_codes
WHERE identifier = $1 AND purpose = $2 AND used_at IS NULL AND expires_at > NOW()
ORDER BY created_at DESC
LIMIT 1
FOR UPDATE SKIP LOCKED;

-- name: MarkOTPUsed :exec
UPDATE otp_codes SET used_at = NOW() WHERE id = $1;

-- name: IncrementOTPAttempts :exec
UPDATE otp_codes SET attempts = attempts + 1 WHERE id = $1;

-- name: DeleteExpiredOTPs :exec
DELETE FROM otp_codes WHERE expires_at < NOW();

-- name: CreateAuditLog :exec
INSERT INTO audit_logs (id, user_id, event_type, event_data, ip_address, user_agent, success, error_message, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);
