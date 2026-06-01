-- Phase 10: Tax profiles, calculations, document refs, and report jobs

CREATE TABLE IF NOT EXISTS tax_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    entity_id UUID NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    npwp VARCHAR(20),
    tax_status VARCHAR(20) NOT NULL DEFAULT 'NON_PKP',
    is_ppn_liable BOOLEAN NOT NULL DEFAULT FALSE,
    default_ppn_rate NUMERIC(5,4) NOT NULL DEFAULT 0.1100,
    pph23_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    default_pph23_rate NUMERIC(5,4) NOT NULL DEFAULT 0.0200,
    pkp_number VARCHAR(50),
    tax_office_code VARCHAR(10),
    efaktur_ready BOOLEAN NOT NULL DEFAULT FALSE,
    ebupot_ready BOOLEAN NOT NULL DEFAULT FALSE,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_tax_profile_workspace UNIQUE (workspace_id),
    CONSTRAINT chk_tax_status CHECK (tax_status IN ('NON_PKP', 'PKP', 'NOT_REGISTERED'))
);

CREATE INDEX IF NOT EXISTS idx_tax_profiles_entity ON tax_profiles(entity_id);

CREATE TABLE IF NOT EXISTS tax_calculations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    transaction_id UUID NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    tax_type VARCHAR(20) NOT NULL,
    direction VARCHAR(10) NOT NULL,
    base_amount NUMERIC(18,2) NOT NULL,
    tax_rate NUMERIC(5,4) NOT NULL,
    tax_amount NUMERIC(18,2) NOT NULL,
    period VARCHAR(7) NOT NULL,
    status VARCHAR(15) NOT NULL DEFAULT 'ACTIVE',
    counterparty_entity_id UUID REFERENCES entities(id) ON DELETE SET NULL,
    faktur_number VARCHAR(50),
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_tax_type CHECK (
        tax_type IN ('PPN_MASUKAN', 'PPN_KELUARAN', 'PPH21', 'PPH23', 'PPH4_AYAT2', 'PPH25', 'PPH29')
    ),
    CONSTRAINT chk_tax_direction CHECK (direction IN ('INPUT', 'OUTPUT', 'WITHHOLDING')),
    CONSTRAINT chk_tax_calc_status CHECK (status IN ('ACTIVE', 'REVERSED', 'VOIDED')),
    CONSTRAINT uq_tax_calc_tx_type UNIQUE (transaction_id, tax_type)
);

CREATE INDEX IF NOT EXISTS idx_tax_calculations_workspace_period ON tax_calculations(workspace_id, period);
CREATE INDEX IF NOT EXISTS idx_tax_calculations_transaction ON tax_calculations(transaction_id);
CREATE INDEX IF NOT EXISTS idx_tax_calculations_type ON tax_calculations(workspace_id, tax_type, period);

CREATE TABLE IF NOT EXISTS tax_document_refs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tax_calculation_id UUID NOT NULL REFERENCES tax_calculations(id) ON DELETE CASCADE,
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    ref_type VARCHAR(30) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_tax_doc_ref_type CHECK (
        ref_type IN ('FAKTUR_PAJAK', 'BUKTI_POTONG', 'INVOICE', 'RECEIPT', 'OTHER')
    ),
    CONSTRAINT uq_tax_doc_ref UNIQUE (tax_calculation_id, document_id)
);

CREATE INDEX IF NOT EXISTS idx_tax_document_refs_calc ON tax_document_refs(tax_calculation_id);

CREATE TABLE IF NOT EXISTS tax_report_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    report_type VARCHAR(30) NOT NULL,
    period_from VARCHAR(7) NOT NULL,
    period_to VARCHAR(7) NOT NULL,
    status VARCHAR(15) NOT NULL DEFAULT 'PENDING',
    result JSONB,
    error_message TEXT,
    requested_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,

    CONSTRAINT chk_tax_report_type CHECK (report_type IN ('PPN_SUMMARY', 'PPH_SUMMARY', 'TAX_OVERVIEW')),
    CONSTRAINT chk_tax_report_status CHECK (status IN ('PENDING', 'PROCESSING', 'COMPLETED', 'FAILED'))
);

CREATE INDEX IF NOT EXISTS idx_tax_report_jobs_workspace ON tax_report_jobs(workspace_id, created_at DESC);
