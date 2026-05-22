-- Migration 014: Accounting Core
-- Tables: accounts, items, transactions, transaction_line_items,
--         journal_entries, ledger_entries, account_balances

-- ============================================================
-- ACCOUNTS (Chart of Accounts per workspace)
-- ============================================================
CREATE TABLE IF NOT EXISTS accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    parent_id UUID REFERENCES accounts(id) ON DELETE SET NULL,
    code VARCHAR(10) NOT NULL,
    name VARCHAR(100) NOT NULL,
    account_type VARCHAR(20) NOT NULL,
    normal_balance VARCHAR(6) NOT NULL,
    level INT NOT NULL DEFAULT 1,
    is_system BOOLEAN NOT NULL DEFAULT false,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_account_type CHECK (
        account_type IN ('ASSET', 'LIABILITY', 'EQUITY', 'REVENUE', 'EXPENSE')
    ),
    CONSTRAINT chk_normal_balance CHECK (
        normal_balance IN ('DEBIT', 'CREDIT')
    ),
    CONSTRAINT uq_accounts_workspace_code UNIQUE (workspace_id, code)
);

CREATE INDEX IF NOT EXISTS idx_accounts_workspace ON accounts(workspace_id);
CREATE INDEX IF NOT EXISTS idx_accounts_parent ON accounts(parent_id) WHERE parent_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_accounts_type ON accounts(workspace_id, account_type);
CREATE INDEX IF NOT EXISTS idx_accounts_active ON accounts(workspace_id, is_active) WHERE is_active = true;

-- ============================================================
-- ITEMS (Products/Services per workspace)
-- ============================================================
CREATE TABLE IF NOT EXISTS items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    name VARCHAR(200) NOT NULL,
    item_type VARCHAR(20) NOT NULL,
    unit VARCHAR(20) NOT NULL,
    unit_price NUMERIC(15,2) NOT NULL DEFAULT 0,
    account_id UUID REFERENCES accounts(id) ON DELETE SET NULL,
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_item_type CHECK (
        item_type IN ('BARANG', 'JASA', 'PROYEK', 'AHSP_RAKITAN')
    )
);

CREATE INDEX IF NOT EXISTS idx_items_workspace ON items(workspace_id);
CREATE INDEX IF NOT EXISTS idx_items_active ON items(workspace_id, is_active) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_items_type ON items(workspace_id, item_type);

-- ============================================================
-- TRANSACTIONS (Header - one per business event)
-- ============================================================
CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    transaction_number VARCHAR(50) NOT NULL,
    transaction_type VARCHAR(20) NOT NULL,
    input_mode VARCHAR(10) NOT NULL DEFAULT 'SIMPLE',
    status VARCHAR(10) NOT NULL DEFAULT 'DRAFT',

    -- Counterparty
    counterparty_entity_id UUID REFERENCES entities(id) ON DELETE SET NULL,
    counterparty_name VARCHAR(200),

    -- Core data
    description TEXT,
    transaction_date DATE NOT NULL,
    amount NUMERIC(15,2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'IDR',

    -- AI categorization
    category VARCHAR(50),
    ai_confidence NUMERIC(3,2),

    -- Payment context (for SALES/PURCHASE)
    payment_method VARCHAR(10),
    includes_tax BOOLEAN NOT NULL DEFAULT false,
    tax_amount NUMERIC(15,2) NOT NULL DEFAULT 0,

    -- Reversal reference
    reversed_transaction_id UUID REFERENCES transactions(id) ON DELETE SET NULL,

    -- Audit
    created_by UUID NOT NULL REFERENCES users(id),
    posted_at TIMESTAMPTZ,
    voided_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_tx_type CHECK (
        transaction_type IN ('CASH_IN', 'CASH_OUT', 'SALES', 'PURCHASE', 'JOURNAL', 'REVERSAL')
    ),
    CONSTRAINT chk_tx_mode CHECK (
        input_mode IN ('SIMPLE', 'ADVANCED', 'OCR')
    ),
    CONSTRAINT chk_tx_status CHECK (
        status IN ('DRAFT', 'POSTED', 'VOID', 'FAILED')
    ),
    CONSTRAINT chk_payment_method CHECK (
        payment_method IS NULL OR payment_method IN ('TUNAI', 'KREDIT', 'TRANSFER')
    ),
    CONSTRAINT uq_tx_workspace_number UNIQUE (workspace_id, transaction_number)
);

CREATE INDEX IF NOT EXISTS idx_tx_workspace ON transactions(workspace_id);
CREATE INDEX IF NOT EXISTS idx_tx_workspace_date ON transactions(workspace_id, transaction_date DESC);
CREATE INDEX IF NOT EXISTS idx_tx_workspace_status ON transactions(workspace_id, status);
CREATE INDEX IF NOT EXISTS idx_tx_workspace_type ON transactions(workspace_id, transaction_type);
CREATE INDEX IF NOT EXISTS idx_tx_counterparty ON transactions(counterparty_entity_id) WHERE counterparty_entity_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_tx_reversed ON transactions(reversed_transaction_id) WHERE reversed_transaction_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_tx_created_by ON transactions(created_by);

-- ============================================================
-- TRANSACTION LINE ITEMS (Multi-item support per transaction)
-- ============================================================
CREATE TABLE IF NOT EXISTS transaction_line_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id UUID NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    workspace_id UUID NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    item_id UUID REFERENCES items(id) ON DELETE SET NULL,
    description VARCHAR(300) NOT NULL,
    quantity NUMERIC(12,4) NOT NULL DEFAULT 1,
    unit VARCHAR(20) NOT NULL DEFAULT 'Pcs',
    unit_price NUMERIC(15,2) NOT NULL,
    discount_amount NUMERIC(15,2) NOT NULL DEFAULT 0,
    tax_amount NUMERIC(15,2) NOT NULL DEFAULT 0,
    line_total NUMERIC(15,2) NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_line_items_tx ON transaction_line_items(transaction_id);
CREATE INDEX IF NOT EXISTS idx_line_items_workspace ON transaction_line_items(workspace_id);
CREATE INDEX IF NOT EXISTS idx_line_items_item ON transaction_line_items(item_id) WHERE item_id IS NOT NULL;

-- ============================================================
-- JOURNAL ENTRIES (Double-entry lines per transaction)
-- ============================================================
CREATE TABLE IF NOT EXISTS journal_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id UUID NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    workspace_id UUID NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    account_id UUID NOT NULL REFERENCES accounts(id),
    description TEXT,
    debit NUMERIC(15,2) NOT NULL DEFAULT 0,
    credit NUMERIC(15,2) NOT NULL DEFAULT 0,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_debit_or_credit CHECK (
        (debit > 0 AND credit = 0) OR (debit = 0 AND credit > 0)
    )
);

CREATE INDEX IF NOT EXISTS idx_journal_tx ON journal_entries(transaction_id);
CREATE INDEX IF NOT EXISTS idx_journal_workspace ON journal_entries(workspace_id);
CREATE INDEX IF NOT EXISTS idx_journal_account ON journal_entries(account_id);
CREATE INDEX IF NOT EXISTS idx_journal_workspace_account ON journal_entries(workspace_id, account_id);

-- ============================================================
-- LEDGER ENTRIES (Posted entries with running balance - written by worker)
-- ============================================================
CREATE TABLE IF NOT EXISTS ledger_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    account_id UUID NOT NULL REFERENCES accounts(id),
    journal_entry_id UUID NOT NULL REFERENCES journal_entries(id) ON DELETE CASCADE,
    transaction_id UUID NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    transaction_date DATE NOT NULL,
    debit NUMERIC(15,2) NOT NULL DEFAULT 0,
    credit NUMERIC(15,2) NOT NULL DEFAULT 0,
    running_balance NUMERIC(15,2) NOT NULL,
    posted_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ledger_workspace ON ledger_entries(workspace_id);
CREATE INDEX IF NOT EXISTS idx_ledger_account ON ledger_entries(workspace_id, account_id, posted_at DESC);
CREATE INDEX IF NOT EXISTS idx_ledger_tx ON ledger_entries(transaction_id);
CREATE INDEX IF NOT EXISTS idx_ledger_date ON ledger_entries(workspace_id, transaction_date DESC);

-- ============================================================
-- ACCOUNT BALANCES (Period summary - upserted by ledger worker)
-- ============================================================
CREATE TABLE IF NOT EXISTS account_balances (
    workspace_id UUID NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    period VARCHAR(7) NOT NULL,
    total_debit NUMERIC(15,2) NOT NULL DEFAULT 0,
    total_credit NUMERIC(15,2) NOT NULL DEFAULT 0,
    ending_balance NUMERIC(15,2) NOT NULL DEFAULT 0,
    transaction_count INT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (workspace_id, account_id, period)
);

CREATE INDEX IF NOT EXISTS idx_balances_workspace_period ON account_balances(workspace_id, period);
CREATE INDEX IF NOT EXISTS idx_balances_account ON account_balances(account_id, period);
