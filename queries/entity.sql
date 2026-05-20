-- name: CreateEntity :one
INSERT INTO entities (id, user_id, entity_type, nama_utama, nik_npwp, nomor_wa, alamat_lengkap, is_shadow, status, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: GetEntityByID :one
SELECT * FROM entities WHERE id = $1;

-- name: GetEntityByUserID :one
SELECT * FROM entities WHERE user_id = $1 AND entity_type = 'ORANG_PRIBADI' LIMIT 1;

-- name: ListEntitiesByUserID :many
SELECT * FROM entities WHERE user_id = $1 ORDER BY created_at DESC;

-- name: UpdateEntity :exec
UPDATE entities
SET nama_utama = $2, nik_npwp = $3, nomor_wa = $4, alamat_lengkap = $5, updated_at = NOW()
WHERE id = $1;

-- name: UpdateEntityStatus :exec
UPDATE entities SET status = $2, updated_at = NOW() WHERE id = $1;

-- name: SearchEntitiesByName :many
SELECT * FROM entities
WHERE nama_utama ILIKE '%' || $1 || '%' AND status = 'ACTIVE'
ORDER BY nama_utama ASC
LIMIT $2 OFFSET $3;

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
