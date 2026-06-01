-- Migration 015: Company Identity & Claim Workflow
-- Tables: entity_verification, entity_legal_ids, entity_aliases,
--         company_claims, claim_documents, claim_audit_log
-- Extensions: pg_trgm (fuzzy matching)

-- ============================================================
-- EXTENSIONS
-- ============================================================
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- ============================================================
-- ENTITY VERIFICATION
-- ============================================================
CREATE TABLE IF NOT EXISTS entity_verification (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_id UUID NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'UNVERIFIED',
    verified_by UUID REFERENCES platform_admins(id) ON DELETE SET NULL,
    verified_at TIMESTAMPTZ,
    rejection_reason TEXT,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_verification_status CHECK (
        status IN ('UNVERIFIED', 'PENDING', 'VERIFIED', 'REJECTED')
    ),
    CONSTRAINT uq_entity_verification UNIQUE (entity_id)
);

CREATE INDEX IF NOT EXISTS idx_entity_verification_status ON entity_verification(status);

-- ============================================================
-- ENTITY LEGAL IDENTIFIERS (NPWP, NIB, SIUP, KTP, AKTA)
-- ============================================================
CREATE TABLE IF NOT EXISTS entity_legal_ids (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_id UUID NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    id_type VARCHAR(20) NOT NULL,
    id_value VARCHAR(100) NOT NULL,
    is_verified BOOLEAN NOT NULL DEFAULT FALSE,
    verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_legal_id_type CHECK (
        id_type IN ('NPWP', 'NIB', 'SIUP', 'KTP', 'AKTA')
    ),
    CONSTRAINT uq_entity_legal_id UNIQUE (entity_id, id_type)
);

CREATE INDEX IF NOT EXISTS idx_entity_legal_ids_entity ON entity_legal_ids(entity_id);
CREATE INDEX IF NOT EXISTS idx_entity_legal_ids_value ON entity_legal_ids(id_value);

-- ============================================================
-- NORMALIZED NAME (for fuzzy matching via pg_trgm)
-- ============================================================
ALTER TABLE entities ADD COLUMN IF NOT EXISTS nama_normalized VARCHAR(255);

CREATE INDEX IF NOT EXISTS idx_entities_nama_normalized_trgm
    ON entities USING gin (nama_normalized gin_trgm_ops)
    WHERE nama_normalized IS NOT NULL;

-- ============================================================
-- ENTITY ALIASES
-- ============================================================
CREATE TABLE IF NOT EXISTS entity_aliases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_id UUID NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    alias VARCHAR(255) NOT NULL,
    alias_normalized VARCHAR(255) NOT NULL,
    source VARCHAR(30) NOT NULL DEFAULT 'MANUAL',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_alias_source CHECK (
        source IN ('MANUAL', 'CLAIM', 'COUNTERPARTY', 'SYSTEM')
    )
);

CREATE INDEX IF NOT EXISTS idx_entity_aliases_entity ON entity_aliases(entity_id);
CREATE INDEX IF NOT EXISTS idx_entity_aliases_normalized_trgm
    ON entity_aliases USING gin (alias_normalized gin_trgm_ops);

-- ============================================================
-- COMPANY CLAIMS
-- ============================================================
CREATE TABLE IF NOT EXISTS company_claims (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_id UUID NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    claimant_user_id UUID NOT NULL REFERENCES users(id),
    claimant_entity_id UUID NOT NULL REFERENCES entities(id),
    status VARCHAR(20) NOT NULL DEFAULT 'DRAFT',
    reviewer_id UUID REFERENCES platform_admins(id) ON DELETE SET NULL,
    reviewed_at TIMESTAMPTZ,
    rejection_reason TEXT,
    dispute_reason TEXT,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_claim_status CHECK (
        status IN ('DRAFT', 'SUBMITTED', 'UNDER_REVIEW', 'APPROVED', 'REJECTED', 'DISPUTED')
    )
);

CREATE INDEX IF NOT EXISTS idx_claims_entity ON company_claims(entity_id);
CREATE INDEX IF NOT EXISTS idx_claims_claimant ON company_claims(claimant_user_id);
CREATE INDEX IF NOT EXISTS idx_claims_status ON company_claims(status);
CREATE INDEX IF NOT EXISTS idx_claims_reviewer ON company_claims(reviewer_id) WHERE reviewer_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_claims_created ON company_claims(created_at DESC);

-- ============================================================
-- CLAIM DOCUMENTS
-- ============================================================
CREATE TABLE IF NOT EXISTS claim_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    claim_id UUID NOT NULL REFERENCES company_claims(id) ON DELETE CASCADE,
    document_type VARCHAR(30) NOT NULL,
    file_key VARCHAR(500) NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    file_size BIGINT NOT NULL DEFAULT 0,
    mime_type VARCHAR(100) NOT NULL DEFAULT 'application/pdf',
    upload_status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    uploaded_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_document_type CHECK (
        document_type IN ('NPWP', 'NIB', 'SIUP', 'AKTA_PENDIRIAN', 'AKTA_PERUBAHAN', 'KTP_DIREKTUR', 'SURAT_KUASA', 'OTHER')
    ),
    CONSTRAINT chk_upload_status CHECK (
        upload_status IN ('PENDING', 'UPLOADED', 'VERIFIED', 'REJECTED')
    )
);

CREATE INDEX IF NOT EXISTS idx_claim_docs_claim ON claim_documents(claim_id);

-- ============================================================
-- CLAIM AUDIT LOG
-- ============================================================
CREATE TABLE IF NOT EXISTS claim_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    claim_id UUID NOT NULL REFERENCES company_claims(id) ON DELETE CASCADE,
    actor_id UUID NOT NULL,
    actor_type VARCHAR(10) NOT NULL,
    action VARCHAR(30) NOT NULL,
    old_status VARCHAR(20),
    new_status VARCHAR(20),
    details JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_audit_actor_type CHECK (actor_type IN ('USER', 'ADMIN', 'SYSTEM')),
    CONSTRAINT chk_audit_action CHECK (
        action IN ('CREATED', 'SUBMITTED', 'DOCUMENT_UPLOADED', 'ASSIGNED', 'APPROVED', 'REJECTED', 'DISPUTED', 'RESUBMITTED', 'NOTE_ADDED')
    )
);

CREATE INDEX IF NOT EXISTS idx_claim_audit_claim ON claim_audit_log(claim_id);
CREATE INDEX IF NOT EXISTS idx_claim_audit_created ON claim_audit_log(created_at DESC);

-- ============================================================
-- COUNTERPARTY ALIASES (workspace-scoped custom names)
-- ============================================================
CREATE TABLE IF NOT EXISTS counterparty_aliases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    entity_id UUID NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    custom_alias VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_counterparty_alias UNIQUE (workspace_id, entity_id)
);

CREATE INDEX IF NOT EXISTS idx_counterparty_aliases_workspace ON counterparty_aliases(workspace_id);
CREATE INDEX IF NOT EXISTS idx_counterparty_aliases_entity ON counterparty_aliases(entity_id);
