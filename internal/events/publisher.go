package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"codeberg.org/azzet/azzetbe/internal/db"
)

// Publisher polls outbox_events and publishes to NATS JetStream
// Uses PostgreSQL LISTEN/NOTIFY for real-time triggering
type Publisher struct {
	Queries    *db.Queries
	Pool       *pgxpool.Pool
	NATS       *NATSClient
	BatchSize  int
	RetryDelay []time.Duration // Exponential backoff delays
}

func NewPublisher(pool *pgxpool.Pool, nats *NATSClient) *Publisher {
	return &Publisher{
		Queries:   db.New(pool),
		Pool:      pool,
		NATS:      nats,
		BatchSize: 50,
		RetryDelay: []time.Duration{
			1 * time.Second,
			5 * time.Second,
			30 * time.Second,
			2 * time.Minute,
			10 * time.Minute,
		},
	}
}

// Run starts the publisher with LISTEN/NOTIFY + polling fallback
func (p *Publisher) Run(ctx context.Context) error {
	slog.Info("outbox publisher started")

	// Process any pending events on startup
	p.processBatch(ctx)

	// Listen for new events via PostgreSQL NOTIFY
	conn, err := p.Pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("publisher: failed to acquire connection: %w", err)
	}
	defer conn.Release()

	_, err = conn.Exec(ctx, "LISTEN outbox_events")
	if err != nil {
		return fmt.Errorf("publisher: failed to LISTEN: %w", err)
	}

	slog.Info("publisher listening for outbox_events notifications")

	// Polling fallback ticker (every 5 seconds for missed notifications)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("outbox publisher stopping")
			return nil

		case <-ticker.C:
			// Fallback polling for any missed events
			p.processBatch(ctx)
			// Also process retryable events
			p.processRetries(ctx)

		default:
			// Wait for NOTIFY with timeout
			notification, err := conn.Conn().WaitForNotification(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return nil // Context cancelled, graceful shutdown
				}
				slog.Warn("publisher: notification wait error", "error", err)
				time.Sleep(1 * time.Second)
				continue
			}

			if notification != nil {
				p.processBatch(ctx)
			}
		}
	}
}

// processBatch fetches and publishes pending outbox events
func (p *Publisher) processBatch(ctx context.Context) {
	events, err := p.Queries.GetPendingOutboxEvents(ctx, int32(p.BatchSize))
	if err != nil {
		slog.Error("publisher: failed to fetch pending events", "error", err)
		return
	}

	for _, outboxEvent := range events {
		if err := p.publishEvent(ctx, &outboxEvent); err != nil {
			slog.Error("publisher: failed to publish event",
				"event_id", outboxEvent.ID.String(),
				"event_type", outboxEvent.EventType,
				"error", err,
			)
			// Mark as failed with next retry time
			retryDelay := p.getRetryDelay(int(outboxEvent.RetryCount))
			nextRetry := time.Now().Add(retryDelay)
			_ = p.Queries.MarkOutboxEventFailed(ctx, db.MarkOutboxEventFailedParams{
				ID:           outboxEvent.ID,
				ErrorMessage: pgtype.Text{String: err.Error(), Valid: true},
				NextRetryAt:  &nextRetry,
			})
		} else {
			_ = p.Queries.MarkOutboxEventPublished(ctx, outboxEvent.ID)
		}
	}
}

// processRetries handles failed events that are ready for retry
func (p *Publisher) processRetries(ctx context.Context) {
	events, err := p.Queries.GetRetryableOutboxEvents(ctx, int32(p.BatchSize))
	if err != nil {
		return
	}

	for _, outboxEvent := range events {
		if int(outboxEvent.RetryCount) >= int(outboxEvent.MaxRetries) {
			// Move to DLQ
			_ = p.Queries.MarkOutboxEventDLQ(ctx, db.MarkOutboxEventDLQParams{
				ID:           outboxEvent.ID,
				ErrorMessage: pgtype.Text{String: "max retries exceeded", Valid: true},
			})
			slog.Warn("publisher: event moved to DLQ",
				"event_id", outboxEvent.ID.String(),
				"event_type", outboxEvent.EventType,
			)
			continue
		}

		if err := p.publishEvent(ctx, &outboxEvent); err != nil {
			retryDelay := p.getRetryDelay(int(outboxEvent.RetryCount))
			nextRetry := time.Now().Add(retryDelay)
			_ = p.Queries.MarkOutboxEventFailed(ctx, db.MarkOutboxEventFailedParams{
				ID:           outboxEvent.ID,
				ErrorMessage: pgtype.Text{String: err.Error(), Valid: true},
				NextRetryAt:  &nextRetry,
			})
		} else {
			_ = p.Queries.MarkOutboxEventPublished(ctx, outboxEvent.ID)
		}
	}
}

// publishEvent converts an outbox record to an Event and publishes to NATS
func (p *Publisher) publishEvent(ctx context.Context, outboxEvent *db.OutboxEvent) error {
	event := &Event{
		ID:            outboxEvent.ID.String(),
		Type:          outboxEvent.EventType,
		Version:       int(outboxEvent.EventVersion),
		CorrelationID: outboxEvent.CorrelationID.String(),
		OccurredAt:    outboxEvent.CreatedAt,
		Payload:       outboxEvent.Payload,
	}

	if outboxEvent.WorkspaceID.Valid {
		wsID := uuid.UUID(outboxEvent.WorkspaceID.Bytes).String()
		event.WorkspaceID = &wsID
	}
	if outboxEvent.ActorID.Valid {
		actorID := uuid.UUID(outboxEvent.ActorID.Bytes).String()
		event.ActorID = &actorID
	}
	if outboxEvent.CausationID.Valid {
		causationID := uuid.UUID(outboxEvent.CausationID.Bytes).String()
		event.CausationID = &causationID
	}
	if outboxEvent.IdempotencyKey.Valid {
		event.IdempotencyKey = &outboxEvent.IdempotencyKey.String
	}
	if outboxEvent.Metadata != nil {
		var meta EventMetadata
		if err := json.Unmarshal(outboxEvent.Metadata, &meta); err == nil {
			event.Metadata = &meta
		}
	}

	return p.NATS.Publish(ctx, event)
}

func (p *Publisher) getRetryDelay(retryCount int) time.Duration {
	if retryCount < len(p.RetryDelay) {
		return p.RetryDelay[retryCount]
	}
	return p.RetryDelay[len(p.RetryDelay)-1]
}

// Cleanup removes old published events (called by scheduled job)
func (p *Publisher) Cleanup(ctx context.Context, olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)
	return p.Queries.CleanupPublishedEvents(ctx, &cutoff)
}
