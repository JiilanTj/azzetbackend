-- ============================================================
-- Entity Verification
-- ============================================================

-- name: CreateEntityVerification :one
INSERT INTO entity_verification (id, entity_id, status, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetEntityVerification :one
SELECT * FROM entity_verification WHERE entity_id = $1;

-- name: UpdateEntityVerificationStatus :exec
UPDATE entity_verification
SET status = $2, verified_by = $3, verified_at = $4, rejection_reason = $5, notes = $6, updated_at = NOW()
WHERE entity_id = $1;

-- name: ListEntitiesByVerificationStatus :many
SELECT ev.*, e.nama_utama, e.entity_type, e.is_shadow
FROM entity_verification ev
JOIN entities e ON e.id = ev.entity_id
WHERE ev.status = $1
ORDER BY ev.updated_at DESC
LIMIT $2 OFFSET $3;

-- ============================================================
-- Entity Legal IDs
-- ============================================================

-- name: CreateEntityLegalID :one
INSERT INTO entity_legal_ids (id, entity_id, id_type, id_value, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetEntityLegalIDs :many
SELECT * FROM entity_legal_ids WHERE entity_id = $1 ORDER BY id_type;

-- name: GetEntityByLegalID :one
SELECT e.* FROM entities e
JOIN entity_legal_ids eli ON eli.entity_id = e.id
WHERE eli.id_type = $1 AND eli.id_value = $2;

-- name: UpdateEntityLegalID :exec
UPDATE entity_legal_ids
SET id_value = $3, updated_at = NOW()
WHERE entity_id = $1 AND id_type = $2;

-- name: VerifyEntityLegalID :exec
UPDATE entity_legal_ids
SET is_verified = TRUE, verified_at = NOW(), updated_at = NOW()
WHERE entity_id = $1 AND id_type = $2;

-- name: DeleteEntityLegalID :exec
DELETE FROM entity_legal_ids WHERE entity_id = $1 AND id_type = $2;

-- ============================================================
-- Entity Aliases
-- ============================================================

-- name: CreateEntityAlias :one
INSERT INTO entity_aliases (id, entity_id, alias, alias_normalized, source, created_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetEntityAliases :many
SELECT * FROM entity_aliases WHERE entity_id = $1 ORDER BY created_at DESC;

-- name: DeleteEntityAlias :exec
DELETE FROM entity_aliases WHERE id = $1 AND entity_id = $2;

-- name: SearchEntitiesByAlias :many
SELECT DISTINCT ON (e.id) e.*, similarity(ea.alias_normalized, $1) as match_score
FROM entities e
JOIN entity_aliases ea ON ea.entity_id = e.id
WHERE ea.alias_normalized % $1
ORDER BY e.id, similarity(ea.alias_normalized, $1) DESC
LIMIT $2;

-- ============================================================
-- Normalized Name & Fuzzy Search
-- ============================================================

-- name: UpdateEntityNormalizedName :exec
UPDATE entities SET nama_normalized = $2, updated_at = NOW() WHERE id = $1;

-- name: SearchEntitiesFuzzy :many
SELECT *, similarity(nama_normalized, $1) as match_score
FROM entities
WHERE nama_normalized % $1 AND status = 'ACTIVE'
ORDER BY similarity(nama_normalized, $1) DESC
LIMIT $2 OFFSET $3;

-- name: FindDuplicateEntities :many
SELECT *, similarity(nama_normalized, $1) as match_score
FROM entities
WHERE nama_normalized % $1
  AND id != $2
  AND status = 'ACTIVE'
  AND similarity(nama_normalized, $1) >= 0.4
ORDER BY similarity(nama_normalized, $1) DESC
LIMIT $3;

-- ============================================================
-- Company Claims
-- ============================================================

-- name: CreateCompanyClaim :one
INSERT INTO company_claims (id, entity_id, claimant_user_id, claimant_entity_id, status, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetCompanyClaimByID :one
SELECT * FROM company_claims WHERE id = $1;

-- name: GetActiveClaimForEntity :one
SELECT * FROM company_claims
WHERE entity_id = $1 AND status NOT IN ('REJECTED', 'APPROVED')
LIMIT 1;

-- name: GetClaimsByClaimant :many
SELECT cc.*, e.nama_utama as entity_name, e.entity_type
FROM company_claims cc
JOIN entities e ON e.id = cc.entity_id
WHERE cc.claimant_user_id = $1
ORDER BY cc.created_at DESC;

-- name: ListClaimsByStatus :many
SELECT cc.*, e.nama_utama as entity_name, e.entity_type, u.name as claimant_name
FROM company_claims cc
JOIN entities e ON e.id = cc.entity_id
JOIN users u ON u.id = cc.claimant_user_id
WHERE cc.status = $1
ORDER BY cc.created_at ASC
LIMIT $2 OFFSET $3;

-- name: ListAllClaims :many
SELECT cc.*, e.nama_utama as entity_name, e.entity_type, u.name as claimant_name
FROM company_claims cc
JOIN entities e ON e.id = cc.entity_id
JOIN users u ON u.id = cc.claimant_user_id
ORDER BY cc.created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdateClaimStatus :exec
UPDATE company_claims
SET status = $2, updated_at = NOW()
WHERE id = $1;

-- name: AssignClaimReviewer :exec
UPDATE company_claims
SET reviewer_id = $2, status = 'UNDER_REVIEW', updated_at = NOW()
WHERE id = $1;

-- name: ApproveCompanyClaim :exec
UPDATE company_claims
SET status = 'APPROVED', reviewer_id = $2, reviewed_at = $3, notes = $4, updated_at = NOW()
WHERE id = $1;

-- name: RejectCompanyClaim :exec
UPDATE company_claims
SET status = 'REJECTED', reviewer_id = $2, reviewed_at = $3, rejection_reason = $4, updated_at = NOW()
WHERE id = $1;

-- name: DisputeCompanyClaim :exec
UPDATE company_claims
SET status = 'DISPUTED', dispute_reason = $2, updated_at = NOW()
WHERE id = $1;

-- name: CountClaimsByStatus :one
SELECT COUNT(*) FROM company_claims WHERE status = $1;

-- ============================================================
-- Claim Documents
-- ============================================================

-- name: CreateClaimDocument :one
INSERT INTO claim_documents (id, claim_id, document_type, file_key, file_name, file_size, mime_type, upload_status, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: GetClaimDocuments :many
SELECT * FROM claim_documents WHERE claim_id = $1 ORDER BY created_at;

-- name: GetClaimDocumentByID :one
SELECT * FROM claim_documents WHERE id = $1;

-- name: MarkDocumentUploaded :exec
UPDATE claim_documents SET upload_status = 'UPLOADED', uploaded_at = NOW() WHERE id = $1;

-- name: UpdateDocumentStatus :exec
UPDATE claim_documents SET upload_status = $2 WHERE id = $1;

-- name: CountClaimDocuments :one
SELECT COUNT(*) FROM claim_documents WHERE claim_id = $1 AND upload_status = 'UPLOADED';

-- ============================================================
-- Claim Audit Log
-- ============================================================

-- name: CreateClaimAuditEntry :exec
INSERT INTO claim_audit_log (id, claim_id, actor_id, actor_type, action, old_status, new_status, details, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: GetClaimAuditLog :many
SELECT * FROM claim_audit_log WHERE claim_id = $1 ORDER BY created_at ASC;

-- ============================================================
-- Counterparty Aliases
-- ============================================================

-- name: CreateCounterpartyAlias :one
INSERT INTO counterparty_aliases (id, workspace_id, entity_id, custom_alias, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetCounterpartyAlias :one
SELECT * FROM counterparty_aliases WHERE workspace_id = $1 AND entity_id = $2;

-- name: ListCounterpartyAliases :many
SELECT ca.*, e.nama_utama as entity_name
FROM counterparty_aliases ca
JOIN entities e ON e.id = ca.entity_id
WHERE ca.workspace_id = $1
ORDER BY ca.custom_alias ASC;

-- name: UpdateCounterpartyAlias :exec
UPDATE counterparty_aliases
SET custom_alias = $3, updated_at = NOW()
WHERE workspace_id = $1 AND entity_id = $2;

-- name: DeleteCounterpartyAlias :exec
DELETE FROM counterparty_aliases WHERE workspace_id = $1 AND entity_id = $2;

-- ============================================================
-- Counterparty Search (privacy-safe)
-- ============================================================

-- name: SearchCounterpartiesFuzzy :many
SELECT e.id, e.nama_utama, e.entity_type, e.is_shadow,
       similarity(e.nama_normalized, $1) as match_score
FROM entities e
WHERE e.nama_normalized % $1
  AND e.status = 'ACTIVE'
  AND e.entity_type = 'BADAN_USAHA'
ORDER BY similarity(e.nama_normalized, $1) DESC
LIMIT $2;

-- name: LinkShadowEntity :exec
UPDATE entities
SET user_id = $2, is_shadow = FALSE, status = 'CLAIMED', updated_at = NOW()
WHERE id = $1 AND is_shadow = TRUE;
