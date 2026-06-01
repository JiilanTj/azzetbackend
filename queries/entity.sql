-- name: CreateEntity :one
INSERT INTO entities (id, user_id, entity_type, nama_utama, nik_npwp, nomor_wa, alamat_lengkap, is_shadow, status, created_at, updated_at, nama_normalized)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING *;

-- name: GetEntityByID :one
SELECT * FROM entities WHERE id = $1;

-- name: GetEntityByUserID :one
SELECT * FROM entities WHERE user_id = $1 AND entity_type = 'ORANG_PRIBADI' LIMIT 1;

-- name: ListEntitiesByUserID :many
SELECT * FROM entities WHERE user_id = $1 ORDER BY created_at DESC;

-- name: UpdateEntity :exec
UPDATE entities
SET nama_utama = $2, nik_npwp = $3, nomor_wa = $4, alamat_lengkap = $5, nama_normalized = $6, updated_at = NOW()
WHERE id = $1;

-- name: UpdateEntityStatus :exec
UPDATE entities SET status = $2, updated_at = NOW() WHERE id = $1;

-- name: SearchEntitiesByName :many
SELECT * FROM entities
WHERE nama_utama ILIKE '%' || $1 || '%' AND status = 'ACTIVE'
ORDER BY nama_utama ASC
LIMIT $2 OFFSET $3;

-- name: UserCanViewEntityAsWorkspaceMember :one
SELECT EXISTS(
    SELECT 1 FROM entity_relations er
    JOIN entities pe ON pe.id = er.subject_id
    WHERE er.object_id = $1
      AND pe.user_id = $2
      AND er.relation_type IN ('PEMILIK', 'KARYAWAN')
      AND er.status = 'ACTIVE'
);

-- name: UserCanViewEntityAsCounterparty :one
SELECT EXISTS(
    SELECT 1 FROM entity_relations cp
    JOIN entity_relations member ON member.object_id = cp.object_id
    JOIN entities pe ON pe.id = member.subject_id
    WHERE cp.subject_id = $1
      AND pe.user_id = $2
      AND cp.relation_type IN ('PELANGGAN', 'VENDOR')
      AND member.relation_type IN ('PEMILIK', 'KARYAWAN')
      AND cp.status = 'ACTIVE'
      AND member.status = 'ACTIVE'
);

-- name: CreateEntityMeta :one
INSERT INTO entity_meta (id, entity_id, bidang_usaha, logo_url, website, email, description, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: GetEntityMetaByEntityID :one
SELECT * FROM entity_meta WHERE entity_id = $1;

-- name: UpdateEntityMeta :exec
UPDATE entity_meta
SET bidang_usaha = $2, logo_url = $3, website = $4, email = $5, description = $6, updated_at = NOW()
WHERE entity_id = $1;

-- name: UpsertEntityMeta :one
INSERT INTO entity_meta (id, entity_id, bidang_usaha, logo_url, website, email, description, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (entity_id)
DO UPDATE SET bidang_usaha = EXCLUDED.bidang_usaha, logo_url = EXCLUDED.logo_url, website = EXCLUDED.website, email = EXCLUDED.email, description = EXCLUDED.description, updated_at = NOW()
RETURNING *;
