-- Entities: Core identity table (lean, fast reads)
CREATE TABLE IF NOT EXISTS entities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    entity_type VARCHAR(20) NOT NULL,
    nama_utama VARCHAR(255) NOT NULL,
    nik_npwp VARCHAR(50),
    nomor_wa VARCHAR(20),
    alamat_lengkap TEXT,
    is_shadow BOOLEAN NOT NULL DEFAULT FALSE,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT check_entity_type CHECK (entity_type IN ('ORANG_PRIBADI', 'BADAN_USAHA')),
    CONSTRAINT check_entity_status CHECK (status IN ('ACTIVE', 'INACTIVE', 'CLAIMED'))
);

CREATE INDEX IF NOT EXISTS idx_entities_user_id ON entities(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_entities_type ON entities(entity_type);
CREATE INDEX IF NOT EXISTS idx_entities_status ON entities(status);
CREATE INDEX IF NOT EXISTS idx_entities_nama ON entities(nama_utama);
CREATE INDEX IF NOT EXISTS idx_entities_shadow ON entities(is_shadow) WHERE is_shadow = TRUE;
CREATE INDEX IF NOT EXISTS idx_entities_nik_npwp ON entities(nik_npwp) WHERE nik_npwp IS NOT NULL;

-- Entity Meta: Heavy administrative data (keeps entities lean)
CREATE TABLE IF NOT EXISTS entity_meta (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_id UUID NOT NULL UNIQUE REFERENCES entities(id) ON DELETE CASCADE,
    bidang_usaha VARCHAR(255),
    logo_url TEXT,
    website VARCHAR(255),
    email VARCHAR(255),
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_entity_meta_entity_id ON entity_meta(entity_id);
