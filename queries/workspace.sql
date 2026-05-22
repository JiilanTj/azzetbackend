-- name: CreateRelation :one
INSERT INTO entity_relations (id, object_id, subject_id, relation_type, custom_alias, status, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetRelationByID :one
SELECT * FROM entity_relations WHERE id = $1;

-- name: GetRelation :one
SELECT * FROM entity_relations
WHERE object_id = $1 AND subject_id = $2 AND relation_type = $3;

-- name: ListRelationsByObject :many
SELECT * FROM entity_relations
WHERE object_id = $1 AND status = 'ACTIVE'
ORDER BY relation_type, created_at DESC;

-- name: ListRelationsByObjectAndType :many
SELECT * FROM entity_relations
WHERE object_id = $1 AND relation_type = $2 AND status = 'ACTIVE'
ORDER BY created_at DESC;

-- name: ListRelationsBySubject :many
SELECT * FROM entity_relations
WHERE subject_id = $1 AND status = 'ACTIVE'
ORDER BY created_at DESC;

-- name: ListWorkspacesBySubject :many
SELECT * FROM entity_relations
WHERE subject_id = $1 AND relation_type IN ('PEMILIK', 'KARYAWAN') AND status = 'ACTIVE'
ORDER BY created_at DESC;

-- name: UpdateRelation :exec
UPDATE entity_relations
SET custom_alias = $2, status = $3, updated_at = NOW()
WHERE id = $1;

-- name: UpdateRelationStatus :exec
UPDATE entity_relations SET status = $2, updated_at = NOW() WHERE id = $1;

-- name: DeleteRelation :exec
DELETE FROM entity_relations WHERE id = $1;

-- name: ExistsRelation :one
SELECT EXISTS(
    SELECT 1 FROM entity_relations
    WHERE object_id = $1 AND subject_id = $2 AND relation_type = $3
);

-- name: GetUserWorkspaceAccess :one
SELECT er.id, er.object_id, er.subject_id, er.relation_type, er.custom_alias, er.status, er.created_at, er.updated_at
FROM entity_relations er
WHERE er.object_id = $1 AND er.subject_id = $2 AND er.status = 'ACTIVE'
AND er.relation_type IN ('PEMILIK', 'KARYAWAN')
LIMIT 1;

-- ============================================================
-- Workspace Roles (ABAC)
-- ============================================================

-- name: CreateWorkspaceRole :one
INSERT INTO workspace_roles (id, workspace_id, name, description, permissions, is_system, created_by, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: GetWorkspaceRoleByID :one
SELECT * FROM workspace_roles WHERE id = $1;

-- name: GetWorkspaceRoleByName :one
SELECT * FROM workspace_roles WHERE workspace_id = $1 AND name = $2;

-- name: ListWorkspaceRoles :many
SELECT * FROM workspace_roles WHERE workspace_id = $1 ORDER BY is_system DESC, name ASC;

-- name: UpdateWorkspaceRole :exec
UPDATE workspace_roles
SET name = $2, description = $3, permissions = $4, updated_at = NOW()
WHERE id = $1;

-- name: DeleteWorkspaceRole :exec
DELETE FROM workspace_roles WHERE id = $1 AND is_system = FALSE;

-- ============================================================
-- Workspace Role Assignments
-- ============================================================

-- name: CreateRoleAssignment :one
INSERT INTO workspace_role_assignments (id, workspace_id, member_entity_id, role_id, assigned_by, created_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetRoleAssignment :one
SELECT * FROM workspace_role_assignments
WHERE workspace_id = $1 AND member_entity_id = $2 AND role_id = $3;

-- name: ListRoleAssignmentsByMember :many
SELECT wra.*, wr.name as role_name, wr.permissions as role_permissions
FROM workspace_role_assignments wra
JOIN workspace_roles wr ON wra.role_id = wr.id
WHERE wra.workspace_id = $1 AND wra.member_entity_id = $2;

-- name: ListRoleAssignmentsByWorkspace :many
SELECT wra.*, wr.name as role_name, wr.permissions as role_permissions
FROM workspace_role_assignments wra
JOIN workspace_roles wr ON wra.role_id = wr.id
WHERE wra.workspace_id = $1
ORDER BY wra.created_at DESC;

-- name: DeleteRoleAssignment :exec
DELETE FROM workspace_role_assignments
WHERE workspace_id = $1 AND member_entity_id = $2 AND role_id = $3;

-- name: DeleteAllRoleAssignmentsForMember :exec
DELETE FROM workspace_role_assignments
WHERE workspace_id = $1 AND member_entity_id = $2;

-- name: GetMemberPermissions :many
SELECT DISTINCT unnest(wr.permissions) as permission
FROM workspace_role_assignments wra
JOIN workspace_roles wr ON wra.role_id = wr.id
WHERE wra.workspace_id = $1 AND wra.member_entity_id = $2;
