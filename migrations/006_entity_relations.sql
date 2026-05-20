-- Master Roles: Role definitions with JSONB permissions
CREATE TABLE IF NOT EXISTS master_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) NOT NULL UNIQUE,
    description TEXT,
    permissions JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed default roles
INSERT INTO master_roles (id, name, description, permissions) VALUES
    (gen_random_uuid(), 'PEMILIK', 'Full access to workspace', '["*"]'),
    (gen_random_uuid(), 'AKUNTAN', 'Accounting and reporting access', '["transaction:*", "report:*", "account:*", "item:read"]'),
    (gen_random_uuid(), 'KASIR', 'Transaction entry only', '["transaction:create", "transaction:read", "item:read"]'),
    (gen_random_uuid(), 'VIEWER', 'Read-only access', '["transaction:read", "report:read", "account:read", "item:read"]');

-- Entity Relations: Multi-tenant isolation core
CREATE TABLE IF NOT EXISTS entity_relations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    object_id UUID NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    subject_id UUID NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    relation_type VARCHAR(30) NOT NULL,
    custom_alias VARCHAR(255),
    role_id UUID REFERENCES master_roles(id) ON DELETE SET NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT check_relation_type CHECK (
        relation_type IN ('PEMILIK', 'KARYAWAN', 'PELANGGAN', 'VENDOR')
    ),
    CONSTRAINT check_relation_status CHECK (status IN ('ACTIVE', 'INACTIVE')),
    CONSTRAINT uq_object_subject_type UNIQUE (object_id, subject_id, relation_type)
);

CREATE INDEX IF NOT EXISTS idx_relations_object_id ON entity_relations(object_id);
CREATE INDEX IF NOT EXISTS idx_relations_subject_id ON entity_relations(subject_id);
CREATE INDEX IF NOT EXISTS idx_relations_type ON entity_relations(relation_type);
CREATE INDEX IF NOT EXISTS idx_relations_status ON entity_relations(status);
CREATE INDEX IF NOT EXISTS idx_relations_object_type ON entity_relations(object_id, relation_type) WHERE status = 'ACTIVE';
