-- name: GetTaxProfileByWorkspace :one
SELECT * FROM tax_profiles WHERE workspace_id = $1;

-- name: CreateTaxProfile :one
INSERT INTO tax_profiles (
    id, workspace_id, entity_id, npwp, tax_status, is_ppn_liable,
    default_ppn_rate, pph23_enabled, default_pph23_rate,
    pkp_number, tax_office_code, efaktur_ready, ebupot_ready, notes,
    created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
) RETURNING *;

-- name: UpdateTaxProfile :one
UPDATE tax_profiles SET
    npwp = $2,
    tax_status = $3,
    is_ppn_liable = $4,
    default_ppn_rate = $5,
    pph23_enabled = $6,
    default_pph23_rate = $7,
    pkp_number = $8,
    tax_office_code = $9,
    efaktur_ready = $10,
    ebupot_ready = $11,
    notes = $12,
    updated_at = NOW()
WHERE workspace_id = $1
RETURNING *;

-- name: CreateTaxCalculation :one
INSERT INTO tax_calculations (
    id, workspace_id, transaction_id, tax_type, direction,
    base_amount, tax_rate, tax_amount, period, status,
    counterparty_entity_id, faktur_number, metadata, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
) RETURNING *;

-- name: GetTaxCalculationByID :one
SELECT * FROM tax_calculations WHERE id = $1 AND workspace_id = $2;

-- name: GetTaxCalculationByTransactionAndType :one
SELECT * FROM tax_calculations
WHERE transaction_id = $1 AND tax_type = $2;

-- name: ListTaxCalculations :many
SELECT tc.*, t.transaction_number, t.transaction_type, t.description AS transaction_description
FROM tax_calculations tc
JOIN transactions t ON t.id = tc.transaction_id
WHERE tc.workspace_id = $1
  AND ($2::text IS NULL OR tc.period = $2)
  AND ($3::text IS NULL OR tc.tax_type = $3)
  AND ($4::text IS NULL OR tc.status = $4)
ORDER BY tc.created_at DESC
LIMIT $5 OFFSET $6;

-- name: CountTaxCalculations :one
SELECT COUNT(*) FROM tax_calculations
WHERE workspace_id = $1
  AND ($2::text IS NULL OR period = $2)
  AND ($3::text IS NULL OR tax_type = $3)
  AND ($4::text IS NULL OR status = $4);

-- name: VoidTaxCalculationsByTransaction :exec
UPDATE tax_calculations
SET status = 'VOIDED', updated_at = NOW()
WHERE transaction_id = $1 AND workspace_id = $2 AND status = 'ACTIVE';

-- name: GetPPNSummaryByPeriod :one
SELECT
    COALESCE(SUM(CASE WHEN tax_type = 'PPN_MASUKAN' THEN tax_amount ELSE 0 END), 0)::numeric AS ppn_masukan,
    COALESCE(SUM(CASE WHEN tax_type = 'PPN_KELUARAN' THEN tax_amount ELSE 0 END), 0)::numeric AS ppn_keluaran,
    COALESCE(SUM(CASE WHEN tax_type = 'PPN_MASUKAN' THEN base_amount ELSE 0 END), 0)::numeric AS dpp_masukan,
    COALESCE(SUM(CASE WHEN tax_type = 'PPN_KELUARAN' THEN base_amount ELSE 0 END), 0)::numeric AS dpp_keluaran,
    COUNT(*) FILTER (WHERE tax_type IN ('PPN_MASUKAN', 'PPN_KELUARAN'))::bigint AS transaction_count
FROM tax_calculations
WHERE workspace_id = $1
  AND period = $2
  AND status = 'ACTIVE';

-- name: GetPPhSummaryByPeriod :many
SELECT
    tax_type,
    direction,
    COALESCE(SUM(base_amount), 0)::numeric AS total_base,
    COALESCE(SUM(tax_amount), 0)::numeric AS total_tax,
    COUNT(*)::bigint AS count
FROM tax_calculations
WHERE workspace_id = $1
  AND period >= $2
  AND period <= $3
  AND tax_type LIKE 'PPH%'
  AND status = 'ACTIVE'
GROUP BY tax_type, direction
ORDER BY tax_type;

-- name: CreateTaxDocumentRef :one
INSERT INTO tax_document_refs (id, tax_calculation_id, document_id, ref_type, created_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListTaxDocumentRefs :many
SELECT tdr.*, d.file_name, d.document_type, d.mime_type
FROM tax_document_refs tdr
JOIN documents d ON d.id = tdr.document_id
WHERE tdr.tax_calculation_id = $1
ORDER BY tdr.created_at DESC;

-- name: CreateTaxReportJob :one
INSERT INTO tax_report_jobs (
    id, workspace_id, report_type, period_from, period_to,
    status, requested_by, created_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetTaxReportJob :one
SELECT * FROM tax_report_jobs WHERE id = $1 AND workspace_id = $2;

-- name: UpdateTaxReportJobProcessing :exec
UPDATE tax_report_jobs
SET status = 'PROCESSING'
WHERE id = $1 AND status = 'PENDING';

-- name: CompleteTaxReportJob :exec
UPDATE tax_report_jobs
SET status = 'COMPLETED', result = $2, completed_at = NOW()
WHERE id = $1;

-- name: FailTaxReportJob :exec
UPDATE tax_report_jobs
SET status = 'FAILED', error_message = $2, completed_at = NOW()
WHERE id = $1;

-- name: ListTaxReportJobs :many
SELECT * FROM tax_report_jobs
WHERE workspace_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;
