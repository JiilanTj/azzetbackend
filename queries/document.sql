-- name: CreateDocument :one
INSERT INTO documents (
    id, workspace_id, document_type, file_key, file_name, file_size, mime_type,
    upload_status, extraction_status, verification_status, created_by, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
) RETURNING *;

-- name: GetDocumentByID :one
SELECT * FROM documents WHERE id = $1 AND workspace_id = $2;

-- name: ListDocumentsByWorkspace :many
SELECT * FROM documents
WHERE workspace_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountDocumentsByWorkspace :one
SELECT COUNT(*) FROM documents WHERE workspace_id = $1;

-- name: MarkWorkspaceDocumentUploaded :exec
UPDATE documents
SET upload_status = 'UPLOADED', uploaded_at = NOW(), updated_at = NOW()
WHERE id = $1 AND workspace_id = $2;

-- name: UpdateDocumentExtraction :exec
UPDATE documents
SET extraction_status = $3,
    extracted_data = $4,
    extraction_confidence = $5,
    extraction_error = $6,
    processed_at = NOW(),
    updated_at = NOW()
WHERE id = $1 AND workspace_id = $2;

-- name: LinkDocumentTransaction :exec
UPDATE documents
SET transaction_id = $3, updated_at = NOW()
WHERE id = $1 AND workspace_id = $2;

-- name: SetDocumentExtractionProcessing :exec
UPDATE documents
SET extraction_status = 'PROCESSING', updated_at = NOW()
WHERE id = $1 AND workspace_id = $2 AND extraction_status = 'PENDING';

-- name: UpdateDocumentVerification :exec
UPDATE documents
SET verification_status = $3, updated_at = NOW()
WHERE id = $1 AND workspace_id = $2;
