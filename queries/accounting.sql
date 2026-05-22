-- ============================================================
-- ACCOUNTS (Chart of Accounts)
-- ============================================================

-- name: CreateAccount :one
INSERT INTO accounts (id, workspace_id, parent_id, code, name, account_type, normal_balance, level, is_system, is_active, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING *;

-- name: GetAccountByID :one
SELECT * FROM accounts WHERE id = $1 AND workspace_id = $2;

-- name: GetAccountByCode :one
SELECT * FROM accounts WHERE workspace_id = $1 AND code = $2;

-- name: ListAccountsByWorkspace :many
SELECT * FROM accounts
WHERE workspace_id = $1 AND is_active = true
ORDER BY code ASC;

-- name: ListAccountsByType :many
SELECT * FROM accounts
WHERE workspace_id = $1 AND account_type = $2 AND is_active = true
ORDER BY code ASC;

-- name: UpdateAccount :exec
UPDATE accounts SET name = $3, parent_id = $4, is_active = $5, updated_at = NOW()
WHERE id = $1 AND workspace_id = $2 AND is_system = false;

-- name: CountAccountsByWorkspace :one
SELECT COUNT(*) FROM accounts WHERE workspace_id = $1;

-- ============================================================
-- ITEMS
-- ============================================================

-- name: CreateItem :one
INSERT INTO items (id, workspace_id, name, item_type, unit, unit_price, account_id, description, is_active, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: GetItemByID :one
SELECT * FROM items WHERE id = $1 AND workspace_id = $2;

-- name: ListItemsByWorkspace :many
SELECT * FROM items
WHERE workspace_id = $1 AND is_active = true
ORDER BY name ASC
LIMIT $2 OFFSET $3;

-- name: ListItemsByType :many
SELECT * FROM items
WHERE workspace_id = $1 AND item_type = $2 AND is_active = true
ORDER BY name ASC
LIMIT $3 OFFSET $4;

-- name: UpdateItem :exec
UPDATE items SET name = $3, item_type = $4, unit = $5, unit_price = $6, account_id = $7, description = $8, updated_at = NOW()
WHERE id = $1 AND workspace_id = $2;

-- name: SoftDeleteItem :exec
UPDATE items SET is_active = false, updated_at = NOW()
WHERE id = $1 AND workspace_id = $2;

-- name: CountItemsByWorkspace :one
SELECT COUNT(*) FROM items WHERE workspace_id = $1 AND is_active = true;

-- ============================================================
-- TRANSACTIONS
-- ============================================================

-- name: CreateTransaction :one
INSERT INTO transactions (id, workspace_id, transaction_number, transaction_type, input_mode, status, counterparty_entity_id, counterparty_name, description, transaction_date, amount, currency, category, ai_confidence, payment_method, includes_tax, tax_amount, reversed_transaction_id, created_by, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
RETURNING *;

-- name: GetTransactionByID :one
SELECT * FROM transactions WHERE id = $1 AND workspace_id = $2;

-- name: ListTransactionsByWorkspace :many
SELECT * FROM transactions
WHERE workspace_id = $1
ORDER BY transaction_date DESC, created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListTransactionsByStatus :many
SELECT * FROM transactions
WHERE workspace_id = $1 AND status = $2
ORDER BY transaction_date DESC, created_at DESC
LIMIT $3 OFFSET $4;

-- name: ListTransactionsByType :many
SELECT * FROM transactions
WHERE workspace_id = $1 AND transaction_type = $2
ORDER BY transaction_date DESC, created_at DESC
LIMIT $3 OFFSET $4;

-- name: ListTransactionsByDateRange :many
SELECT * FROM transactions
WHERE workspace_id = $1 AND transaction_date >= $2 AND transaction_date <= $3
ORDER BY transaction_date DESC, created_at DESC
LIMIT $4 OFFSET $5;

-- name: ListTransactionsByCounterparty :many
SELECT * FROM transactions
WHERE workspace_id = $1 AND counterparty_entity_id = $2
ORDER BY transaction_date DESC, created_at DESC
LIMIT $3 OFFSET $4;

-- name: UpdateTransactionStatus :exec
UPDATE transactions SET status = $3, updated_at = NOW() WHERE id = $1 AND workspace_id = $2;

-- name: MarkTransactionPosted :exec
UPDATE transactions SET status = 'POSTED', posted_at = NOW(), updated_at = NOW() WHERE id = $1 AND workspace_id = $2;

-- name: MarkTransactionVoid :exec
UPDATE transactions SET status = 'VOID', voided_at = NOW(), updated_at = NOW() WHERE id = $1 AND workspace_id = $2;

-- name: MarkTransactionFailed :exec
UPDATE transactions SET status = 'FAILED', updated_at = NOW() WHERE id = $1 AND workspace_id = $2;

-- name: UpdateDraftTransaction :exec
UPDATE transactions SET description = $3, transaction_date = $4, amount = $5, category = $6, counterparty_entity_id = $7, counterparty_name = $8, payment_method = $9, includes_tax = $10, tax_amount = $11, updated_at = NOW()
WHERE id = $1 AND workspace_id = $2 AND status = 'DRAFT';

-- name: GetNextTransactionNumber :one
SELECT COALESCE(MAX(CAST(SUBSTRING(transaction_number FROM '[0-9]+$') AS INT)), 0) + 1 AS next_number
FROM transactions WHERE workspace_id = $1;

-- name: CountTransactionsByWorkspace :one
SELECT COUNT(*) FROM transactions WHERE workspace_id = $1;

-- ============================================================
-- TRANSACTION LINE ITEMS
-- ============================================================

-- name: CreateTransactionLineItem :one
INSERT INTO transaction_line_items (id, transaction_id, workspace_id, item_id, description, quantity, unit, unit_price, discount_amount, tax_amount, line_total, sort_order, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
RETURNING *;

-- name: ListLineItemsByTransaction :many
SELECT * FROM transaction_line_items
WHERE transaction_id = $1 AND workspace_id = $2
ORDER BY sort_order ASC;

-- name: DeleteLineItemsByTransaction :exec
DELETE FROM transaction_line_items WHERE transaction_id = $1 AND workspace_id = $2;

-- ============================================================
-- JOURNAL ENTRIES
-- ============================================================

-- name: CreateJournalEntry :one
INSERT INTO journal_entries (id, transaction_id, workspace_id, account_id, description, debit, credit, sort_order, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: ListJournalEntriesByTransaction :many
SELECT je.*, a.code as account_code, a.name as account_name
FROM journal_entries je
JOIN accounts a ON je.account_id = a.id
WHERE je.transaction_id = $1 AND je.workspace_id = $2
ORDER BY je.sort_order ASC;

-- name: GetJournalSumByTransaction :one
SELECT COALESCE(SUM(debit), 0) as total_debit, COALESCE(SUM(credit), 0) as total_credit
FROM journal_entries WHERE transaction_id = $1 AND workspace_id = $2;

-- name: DeleteJournalEntriesByTransaction :exec
DELETE FROM journal_entries WHERE transaction_id = $1 AND workspace_id = $2;

-- ============================================================
-- LEDGER ENTRIES
-- ============================================================

-- name: CreateLedgerEntry :one
INSERT INTO ledger_entries (id, workspace_id, account_id, journal_entry_id, transaction_id, transaction_date, debit, credit, running_balance, posted_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: GetLastLedgerBalance :one
SELECT running_balance FROM ledger_entries
WHERE workspace_id = $1 AND account_id = $2
ORDER BY posted_at DESC, id DESC
LIMIT 1;

-- name: ListLedgerEntriesByAccount :many
SELECT le.*, t.transaction_number, t.description as tx_description
FROM ledger_entries le
JOIN transactions t ON le.transaction_id = t.id
WHERE le.workspace_id = $1 AND le.account_id = $2
ORDER BY le.posted_at DESC, le.id DESC
LIMIT $3 OFFSET $4;

-- name: ListLedgerEntriesByAccountDateRange :many
SELECT le.*, t.transaction_number, t.description as tx_description
FROM ledger_entries le
JOIN transactions t ON le.transaction_id = t.id
WHERE le.workspace_id = $1 AND le.account_id = $2 AND le.transaction_date >= $3 AND le.transaction_date <= $4
ORDER BY le.posted_at ASC, le.id ASC;

-- ============================================================
-- ACCOUNT BALANCES
-- ============================================================

-- name: UpsertAccountBalance :one
INSERT INTO account_balances (workspace_id, account_id, period, total_debit, total_credit, ending_balance, transaction_count, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, 1, NOW())
ON CONFLICT (workspace_id, account_id, period)
DO UPDATE SET
    total_debit = account_balances.total_debit + EXCLUDED.total_debit,
    total_credit = account_balances.total_credit + EXCLUDED.total_credit,
    ending_balance = account_balances.ending_balance + EXCLUDED.ending_balance,
    transaction_count = account_balances.transaction_count + 1,
    updated_at = NOW()
RETURNING *;

-- name: GetAccountBalance :one
SELECT * FROM account_balances
WHERE workspace_id = $1 AND account_id = $2 AND period = $3;

-- name: ListAccountBalancesByPeriod :many
SELECT ab.*, a.code as account_code, a.name as account_name, a.account_type, a.normal_balance
FROM account_balances ab
JOIN accounts a ON ab.account_id = a.id
WHERE ab.workspace_id = $1 AND ab.period = $2
ORDER BY a.code ASC;

-- name: ListAccountBalancesByDateRange :many
SELECT ab.*, a.code as account_code, a.name as account_name, a.account_type, a.normal_balance
FROM account_balances ab
JOIN accounts a ON ab.account_id = a.id
WHERE ab.workspace_id = $1 AND ab.period >= $2 AND ab.period <= $3
ORDER BY a.code ASC, ab.period ASC;

-- name: GetTrialBalance :many
SELECT a.id, a.code, a.name, a.account_type, a.normal_balance,
    COALESCE(SUM(ab.total_debit), 0) as total_debit,
    COALESCE(SUM(ab.total_credit), 0) as total_credit,
    COALESCE(SUM(ab.ending_balance), 0) as balance
FROM accounts a
LEFT JOIN account_balances ab ON a.id = ab.account_id AND ab.workspace_id = $1 AND ab.period >= $2 AND ab.period <= $3
WHERE a.workspace_id = $1 AND a.is_active = true
GROUP BY a.id, a.code, a.name, a.account_type, a.normal_balance
HAVING COALESCE(SUM(ab.total_debit), 0) > 0 OR COALESCE(SUM(ab.total_credit), 0) > 0
ORDER BY a.code ASC;

-- name: GetBalanceSheet :many
SELECT a.id, a.code, a.name, a.account_type, a.normal_balance,
    COALESCE(SUM(ab.ending_balance), 0) as balance
FROM accounts a
LEFT JOIN account_balances ab ON a.id = ab.account_id AND ab.workspace_id = $1 AND ab.period <= $2
WHERE a.workspace_id = $1 AND a.is_active = true AND a.account_type IN ('ASSET', 'LIABILITY', 'EQUITY')
GROUP BY a.id, a.code, a.name, a.account_type, a.normal_balance
HAVING COALESCE(SUM(ab.ending_balance), 0) != 0
ORDER BY a.code ASC;

-- name: GetIncomeStatement :many
SELECT a.id, a.code, a.name, a.account_type, a.normal_balance,
    COALESCE(SUM(ab.total_debit), 0) as total_debit,
    COALESCE(SUM(ab.total_credit), 0) as total_credit,
    COALESCE(SUM(ab.ending_balance), 0) as balance
FROM accounts a
LEFT JOIN account_balances ab ON a.id = ab.account_id AND ab.workspace_id = $1 AND ab.period >= $2 AND ab.period <= $3
WHERE a.workspace_id = $1 AND a.is_active = true AND a.account_type IN ('REVENUE', 'EXPENSE')
GROUP BY a.id, a.code, a.name, a.account_type, a.normal_balance
HAVING COALESCE(SUM(ab.ending_balance), 0) != 0
ORDER BY a.code ASC;

-- name: GetCashFlow :many
SELECT le.transaction_date,
    SUM(le.debit) as total_debit,
    SUM(le.credit) as total_credit,
    SUM(le.debit) - SUM(le.credit) as net_flow
FROM ledger_entries le
JOIN accounts a ON le.account_id = a.id
WHERE le.workspace_id = $1 AND a.code IN ('1-1001', '1-1002') AND le.transaction_date >= $2 AND le.transaction_date <= $3
GROUP BY le.transaction_date
ORDER BY le.transaction_date ASC;
