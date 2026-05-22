-- ============================================================
-- Migration 011: ABAC Permission System
-- Replaces master_roles with per-workspace custom roles.
-- ============================================================

-- 1. Create workspace_roles table (custom roles per workspace)
CREATE TABLE IF NOT EXISTS workspace_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    name VARCHAR(50) NOT NULL,
    description TEXT,
    permissions TEXT[] NOT NULL DEFAULT '{}',
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_workspace_role_name UNIQUE (workspace_id, name)
);

CREATE INDEX IF NOT EXISTS idx_workspace_roles_workspace ON workspace_roles(workspace_id);

-- 2. Create workspace_role_assignments table
CREATE TABLE IF NOT EXISTS workspace_role_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    member_entity_id UUID NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES workspace_roles(id) ON DELETE CASCADE,
    assigned_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_workspace_member_role UNIQUE (workspace_id, member_entity_id, role_id)
);

CREATE INDEX IF NOT EXISTS idx_role_assignments_workspace ON workspace_role_assignments(workspace_id);
CREATE INDEX IF NOT EXISTS idx_role_assignments_member ON workspace_role_assignments(member_entity_id);
CREATE INDEX IF NOT EXISTS idx_role_assignments_role ON workspace_role_assignments(role_id);

-- 3. Drop role_id FK from entity_relations (no longer needed)
ALTER TABLE entity_relations DROP CONSTRAINT IF EXISTS entity_relations_role_id_fkey;
ALTER TABLE entity_relations DROP COLUMN IF EXISTS role_id;

-- 4. Drop master_roles table (replaced by workspace_roles)
DROP TABLE IF EXISTS master_roles CASCADE;
