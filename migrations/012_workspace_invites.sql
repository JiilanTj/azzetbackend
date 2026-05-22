-- ============================================================
-- Migration 012: Workspace Invites
-- Email-based invite flow with 24h expiry.
-- ============================================================

CREATE TABLE IF NOT EXISTS workspace_invites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    invited_email VARCHAR(255) NOT NULL,
    role_id UUID NOT NULL REFERENCES workspace_roles(id) ON DELETE CASCADE,
    token VARCHAR(64) NOT NULL UNIQUE,
    invited_by UUID NOT NULL REFERENCES users(id),
    expires_at TIMESTAMPTZ NOT NULL,
    accepted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_workspace_invites_token ON workspace_invites(token);
CREATE INDEX IF NOT EXISTS idx_workspace_invites_email ON workspace_invites(invited_email);
CREATE INDEX IF NOT EXISTS idx_workspace_invites_workspace ON workspace_invites(workspace_id);
