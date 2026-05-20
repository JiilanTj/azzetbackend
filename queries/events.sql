-- name: CreateOutboxEvent :one
INSERT INTO outbox_events (id, event_type, event_version, workspace_id, actor_id, correlation_id, causation_id, idempotency_key, payload, metadata, status, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'pending', $11)
RETURNING *;

-- name: GetPendingOutboxEvents :many
SELECT * FROM outbox_events
WHERE status = 'pending'
ORDER BY created_at ASC
LIMIT $1
FOR UPDATE SKIP LOCKED;

-- name: GetRetryableOutboxEvents :many
SELECT * FROM outbox_events
WHERE status = 'failed' AND retry_count < max_retries AND (next_retry_at IS NULL OR next_retry_at <= NOW())
ORDER BY created_at ASC
LIMIT $1
FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxEventPublished :exec
UPDATE outbox_events SET status = 'published', published_at = NOW() WHERE id = $1;

-- name: MarkOutboxEventFailed :exec
UPDATE outbox_events
SET status = 'failed', retry_count = retry_count + 1, error_message = $2, next_retry_at = $3
WHERE id = $1;

-- name: MarkOutboxEventDLQ :exec
UPDATE outbox_events SET status = 'dlq', error_message = $2 WHERE id = $1;

-- name: CleanupPublishedEvents :exec
DELETE FROM outbox_events WHERE status = 'published' AND published_at < $1;

-- name: GetOutboxEventByID :one
SELECT * FROM outbox_events WHERE id = $1;

-- name: CountPendingOutboxEvents :one
SELECT COUNT(*) FROM outbox_events WHERE status = 'pending';

-- name: IsEventConsumed :one
SELECT EXISTS(SELECT 1 FROM inbox_consumed_events WHERE event_id = $1 AND consumer_name = $2);

-- name: MarkEventConsumed :exec
INSERT INTO inbox_consumed_events (id, event_id, consumer_name, processed_at)
VALUES ($1, $2, $3, NOW())
ON CONFLICT (event_id, consumer_name) DO NOTHING;

-- name: CleanupConsumedEvents :exec
DELETE FROM inbox_consumed_events WHERE processed_at < $1;
