-- ============================================================
-- Migration 013: Add pending_payment status to subscriptions
-- ============================================================

ALTER TABLE tenant_subscriptions DROP CONSTRAINT IF EXISTS check_sub_status;
ALTER TABLE tenant_subscriptions ADD CONSTRAINT check_sub_status CHECK (
    status IN ('active', 'trial', 'expired', 'cancelled', 'pending_payment')
);
