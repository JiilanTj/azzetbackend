-- name: CreateInvoice :one
INSERT INTO invoices (id, workspace_id, subscription_id, invoice_number, amount, currency, status, description, due_date, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: GetInvoiceByID :one
SELECT * FROM invoices WHERE id = $1;

-- name: GetInvoiceByNumber :one
SELECT * FROM invoices WHERE invoice_number = $1;

-- name: ListInvoicesByWorkspace :many
SELECT * FROM invoices WHERE workspace_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: ListPendingInvoices :many
SELECT * FROM invoices WHERE status = 'pending' AND due_date < NOW() ORDER BY due_date ASC;

-- name: UpdateInvoiceStatus :exec
UPDATE invoices SET status = $2, updated_at = NOW() WHERE id = $1;

-- name: MarkInvoicePaid :exec
UPDATE invoices SET status = 'paid', paid_at = NOW(), updated_at = NOW() WHERE id = $1;

-- name: CreatePayment :one
INSERT INTO payments (id, invoice_id, workspace_id, xendit_invoice_id, xendit_payment_url, amount, currency, status, expired_at, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: GetPaymentByID :one
SELECT * FROM payments WHERE id = $1;

-- name: GetPaymentByXenditID :one
SELECT * FROM payments WHERE xendit_invoice_id = $1;

-- name: ListPaymentsByInvoice :many
SELECT * FROM payments WHERE invoice_id = $1 ORDER BY created_at DESC;

-- name: ListPaymentsByWorkspace :many
SELECT * FROM payments WHERE workspace_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: UpdatePaymentStatus :exec
UPDATE payments SET status = $2, payment_method = $3, paid_at = $4, failure_reason = $5, xendit_callback_data = $6, updated_at = NOW()
WHERE id = $1;

-- name: UpdatePaymentExpired :exec
UPDATE payments SET status = 'expired', updated_at = NOW()
WHERE status = 'pending' AND expired_at < NOW();

-- name: ListAllInvoices :many
SELECT i.*, e.nama_utama as workspace_name
FROM invoices i
JOIN entities e ON i.workspace_id = e.id
ORDER BY i.created_at DESC
LIMIT $1 OFFSET $2;
