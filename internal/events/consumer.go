package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go/jetstream"

	"codeberg.org/azzet/azzetbe/internal/db"
)

// Consumer is the base event consumer with idempotency support
type Consumer struct {
	Queries *db.Queries
	Pool    *pgxpool.Pool
	NATS    *NATSClient
}

func NewConsumer(pool *pgxpool.Pool, nats *NATSClient) *Consumer {
	return &Consumer{
		Queries: db.New(pool),
		Pool:    pool,
		NATS:    nats,
	}
}

// EventHandler is a function that processes an event
type EventHandler func(ctx context.Context, event *Event) error

// HandleWithIdempotency wraps an event handler with idempotency check
func (c *Consumer) HandleWithIdempotency(consumerName string, handler EventHandler) func(jetstream.Msg) {
	return func(msg jetstream.Msg) {
		ctx := context.Background()

		// Parse event
		event, err := Unmarshal(msg.Data())
		if err != nil {
			slog.Error("consumer: failed to unmarshal event",
				"consumer", consumerName,
				"error", err,
			)
			msg.Term() // Terminal error, don't retry
			return
		}

		// Idempotency check
		eventID, err := uuid.Parse(event.ID)
		if err != nil {
			msg.Term()
			return
		}

		consumed, err := c.Queries.IsEventConsumed(ctx, db.IsEventConsumedParams{
			EventID:      eventID,
			ConsumerName: consumerName,
		})
		if err != nil {
			slog.Error("consumer: failed to check idempotency",
				"consumer", consumerName,
				"event_id", event.ID,
				"error", err,
			)
			msg.Nak() // Retry later
			return
		}
		if consumed {
			slog.Debug("consumer: event already consumed, skipping",
				"consumer", consumerName,
				"event_id", event.ID,
			)
			msg.Ack()
			return
		}

		// Process event
		if err := handler(ctx, event); err != nil {
			slog.Error("consumer: handler failed",
				"consumer", consumerName,
				"event_id", event.ID,
				"event_type", event.Type,
				"error", err,
			)
			msg.Nak() // Retry (NATS will redeliver based on MaxDeliver)
			return
		}

		// Mark as consumed
		_ = c.Queries.MarkEventConsumed(ctx, db.MarkEventConsumedParams{
			ID:           uuid.New(),
			EventID:      eventID,
			ConsumerName: consumerName,
		})

		msg.Ack()

		slog.Debug("consumer: event processed",
			"consumer", consumerName,
			"event_id", event.ID,
			"event_type", event.Type,
		)
	}
}

// --- Helper: Emit Event from Services ---

// EmitEvent writes an event to the outbox table within a transaction
// This should be called inside the same transaction as business data writes
func EmitEvent(ctx context.Context, tx pgx.Tx, eventType string, payload any, opts ...EventOption) error {
	event, err := NewEvent(eventType, payload, opts...)
	if err != nil {
		return fmt.Errorf("emit: failed to create event: %w", err)
	}

	metadataBytes, _ := json.Marshal(event.Metadata)

	var wsID pgtype.UUID
	if event.WorkspaceID != nil {
		if uid, err := uuid.Parse(*event.WorkspaceID); err == nil {
			wsID = pgtype.UUID{Bytes: uid, Valid: true}
		}
	}

	var actorID *uuid.UUID
	if event.ActorID != nil {
		if uid, err := uuid.Parse(*event.ActorID); err == nil {
			actorID = &uid
		}
	}

	var causationID *uuid.UUID
	if event.CausationID != nil {
		if uid, err := uuid.Parse(*event.CausationID); err == nil {
			causationID = &uid
		}
	}

	correlationUUID, _ := uuid.Parse(event.CorrelationID)

	var idempotencyKey pgtype.Text
	if event.IdempotencyKey != nil {
		idempotencyKey = pgtype.Text{String: *event.IdempotencyKey, Valid: true}
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO outbox_events (id, event_type, event_version, workspace_id, actor_id, correlation_id, causation_id, idempotency_key, payload, metadata, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'pending', $11)
	`,
		uuid.MustParse(event.ID),
		event.Type,
		event.Version,
		wsID,
		actorID,
		correlationUUID,
		causationID,
		idempotencyKey,
		event.Payload,
		metadataBytes,
		event.OccurredAt,
	)
	if err != nil {
		return fmt.Errorf("emit: failed to write outbox event: %w", err)
	}

	return nil
}

// EmitEventDirect writes an event to outbox without a transaction (for simple cases)
func EmitEventDirect(ctx context.Context, pool *pgxpool.Pool, eventType string, payload any, opts ...EventOption) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := EmitEvent(ctx, tx, eventType, payload, opts...); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
