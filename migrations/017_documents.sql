-- Phase 9: Workspace documents for OCR receipt/invoice processing

CREATE TABLE IF NOT EXISTS documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    document_type VARCHAR(30) NOT NULL,
    file_key VARCHAR(500) NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    file_size BIGINT NOT NULL DEFAULT 0,
    mime_type VARCHAR(100) NOT NULL DEFAULT 'application/pdf',
    upload_status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    extraction_status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    verification_status VARCHAR(20) NOT NULL DEFAULT 'UNVERIFIED',
    extracted_data JSONB,
    extraction_confidence NUMERIC(5,4),
    extraction_error TEXT,
    transaction_id UUID REFERENCES transactions(id) ON DELETE SET NULL,
    created_by UUID NOT NULL REFERENCES users(id),
    uploaded_at TIMESTAMPTZ,
    processed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_doc_type CHECK (
        document_type IN ('RECEIPT', 'INVOICE', 'FAKTUR', 'OTHER')
    ),
    CONSTRAINT chk_doc_upload_status CHECK (
        upload_status IN ('PENDING', 'UPLOADED', 'FAILED')
    ),
    CONSTRAINT chk_doc_extraction_status CHECK (
        extraction_status IN ('PENDING', 'PROCESSING', 'COMPLETED', 'FAILED', 'SKIPPED')
    ),
    CONSTRAINT chk_doc_verification_status CHECK (
        verification_status IN ('UNVERIFIED', 'VERIFIED', 'REJECTED')
    )
);

CREATE INDEX IF NOT EXISTS idx_documents_workspace ON documents(workspace_id);
CREATE INDEX IF NOT EXISTS idx_documents_workspace_created ON documents(workspace_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_documents_transaction ON documents(transaction_id) WHERE transaction_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_documents_extraction ON documents(workspace_id, extraction_status);
