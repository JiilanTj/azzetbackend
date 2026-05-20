-- name: CreateRelation :one
INSERT INTO entity_relations (id, object_id, subject_id, relation_type, custom_alias, role_id, status, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
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
SET custom_alias = $2, role_id = $3, status = $4, updated_at = NOW()
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

-- name: GetUserWorkspaceRole :one
SELECT er.*, mr.name as role_name, mr.permissions as role_permissions
FROM entity_relations er
LEFT JOIN master_roles mr ON er.role_id = mr.id
WHERE er.object_id = $1 AND er.subject_id = $2 AND er.status = 'ACTIVE'
AND er.relation_type IN ('PEMILIK', 'KARYAWAN')
LIMIT 1;

-- name: ListRoles :many
SELECT * FROM master_roles ORDER BY name ASC;

-- name: GetRoleByName :one
SELECT * FROM master_roles WHERE name = $1;

-- name: GetRoleByID :one
SELECT * FROM master_roles WHERE id = $1;
