-- Outbox Events: Transactional outbox pattern
-- Events are written in the same transaction as business data
CREATE TABLE IF NOT EXISTS outbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(100) NOT NULL,
    event_version INT NOT NULL DEFAULT 1,
    workspace_id UUID REFERENCES entities(id) ON DELETE SET NULL,
    actor_id UUID,
    correlation_id UUID NOT NULL DEFAULT gen_random_uuid(),
    causation_id UUID,
    idempotency_key VARCHAR(255),
    payload JSONB NOT NULL,
    metadata JSONB,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    published_at TIMESTAMPTZ,
    retry_count INT NOT NULL DEFAULT 0,
    max_retries INT NOT NULL DEFAULT 5,
    next_retry_at TIMESTAMPTZ,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT check_outbox_status CHECK (
        status IN ('pending', 'published', 'failed', 'dlq')
    )
);

CREATE INDEX IF NOT EXISTS idx_outbox_pending ON outbox_events(status, created_at)
    WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_outbox_retry ON outbox_events(status, next_retry_at)
    WHERE status = 'failed' AND next_retry_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_outbox_workspace ON outbox_events(workspace_id);
CREATE INDEX IF NOT EXISTS idx_outbox_type ON outbox_events(event_type);
CREATE INDEX IF NOT EXISTS idx_outbox_correlation ON outbox_events(correlation_id);
CREATE INDEX IF NOT EXISTS idx_outbox_created ON outbox_events(created_at);

-- Inbox Consumed Events: Idempotent consumers
-- Prevents duplicate processing of the same event
CREATE TABLE IF NOT EXISTS inbox_consumed_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL,
    consumer_name VARCHAR(100) NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_event_consumer UNIQUE (event_id, consumer_name)
);

CREATE INDEX IF NOT EXISTS idx_inbox_event_id ON inbox_consumed_events(event_id);
CREATE INDEX IF NOT EXISTS idx_inbox_consumer ON inbox_consumed_events(consumer_name);

-- Function + Trigger: Notify on new outbox event (for real-time publishing)
CREATE OR REPLACE FUNCTION notify_outbox_event() RETURNS trigger AS $$
BEGIN
    PERFORM pg_notify('outbox_events', NEW.id::text);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_outbox_notify ON outbox_events;
CREATE TRIGGER trg_outbox_notify
    AFTER INSERT ON outbox_events
    FOR EACH ROW
    EXECUTE FUNCTION notify_outbox_event();

-- Cleanup: Auto-delete published events older than 14 days
-- (Run via scheduled job or cron)
-- DELETE FROM outbox_events WHERE status = 'published' AND published_at < NOW() - INTERVAL '14 days';
