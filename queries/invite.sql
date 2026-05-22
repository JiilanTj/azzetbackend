-- name: CreateInvite :one
INSERT INTO workspace_invites (id, workspace_id, invited_email, role_id, token, invited_by, expires_at, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetInviteByToken :one
SELECT * FROM workspace_invites WHERE token = $1;

-- name: GetInviteByID :one
SELECT * FROM workspace_invites WHERE id = $1;

-- name: ListPendingInvitesByWorkspace :many
SELECT * FROM workspace_invites
WHERE workspace_id = $1 AND accepted_at IS NULL AND expires_at > NOW()
ORDER BY created_at DESC;

-- name: AcceptInvite :exec
UPDATE workspace_invites SET accepted_at = NOW() WHERE id = $1;

-- name: DeleteInvite :exec
DELETE FROM workspace_invites WHERE id = $1;

-- name: ExistsPendingInvite :one
SELECT EXISTS(
    SELECT 1 FROM workspace_invites
    WHERE workspace_id = $1 AND invited_email = $2 AND accepted_at IS NULL AND expires_at > NOW()
);
