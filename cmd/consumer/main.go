package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"codeberg.org/azzet/azzetbe/internal/config"
	"codeberg.org/azzet/azzetbe/internal/database"
	"codeberg.org/azzet/azzetbe/internal/events"
	"codeberg.org/azzet/azzetbe/internal/shared"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	shared.NewLogger(cfg.AppEnv)

	fmt.Println()
	fmt.Printf("  \033[36m╭─── Azzet Consumer ─────────────────────────────────────╮\033[0m\n")
	fmt.Printf("  \033[36m│                                                         │\033[0m\n")
	fmt.Printf("  \033[36m│   \033[1m\033[37mNATS JetStream Event Consumer\033[0m\033[36m                        │\033[0m\n")
	fmt.Printf("  \033[36m│                                                         │\033[0m\n")
	fmt.Printf("  \033[36m│   \033[32m●\033[0m NATS     \033[1m→\033[0m \033[37m%-37s\033[36m│\033[0m\n", cfg.NatsURL)
	fmt.Printf("  \033[36m│   \033[32m●\033[0m Env      \033[1m→\033[0m \033[33m%-37s\033[36m│\033[0m\n", cfg.AppEnv)
	fmt.Printf("  \033[36m│                                                         │\033[0m\n")
	fmt.Printf("  \033[36m╰─────────────────────────────────────────────────────────╯\033[0m\n")
	fmt.Println()

	db, err := database.NewFromEnv(cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	natsClient, err := events.NewNATSClient(cfg.NatsURL)
	if err != nil {
		slog.Error("failed to connect to nats", "error", err)
		os.Exit(1)
	}
	defer natsClient.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Ensure streams exist
	if err := natsClient.EnsureStreams(ctx); err != nil {
		slog.Error("failed to ensure nats streams", "error", err)
		os.Exit(1)
	}

	// Create consumer base
	consumer := events.NewConsumer(db.Pool, natsClient)

	// --- Register Event Handlers ---

	// User stream: handle user.registered (audit + future use)
	// Entity + workspace are created synchronously during registration.
	// This consumer handles additional async tasks (e.g., welcome email, analytics)
	_, err = natsClient.Subscribe(ctx, events.StreamUser, "user-entity-worker",
		consumer.HandleWithIdempotency("user-entity-worker", func(ctx context.Context, event *events.Event) error {
			if event.Type != events.UserRegistered {
				return nil
			}

			var payload struct {
				UserID string `json:"user_id"`
				Name   string `json:"name"`
			}
			if err := json.Unmarshal(event.Payload, &payload); err != nil {
				return fmt.Errorf("failed to parse user.registered payload: %w", err)
			}

			slog.Info("user-entity-worker: user registered event received",
				"user_id", payload.UserID,
				"name", payload.Name,
			)

			// Entity + workspace already created synchronously during registration.
			// This handler is for future async tasks:
			// - Send welcome email (via Asynq)
			// - Track analytics
			// - Notify admin of new signup

			return nil
		}),
	)
	if err != nil {
		slog.Error("failed to subscribe to user stream", "error", err)
		os.Exit(1)
	}

	// Accounting stream: ledger posting
	_, err = natsClient.Subscribe(ctx, events.StreamAccounting, "ledger-worker",
		consumer.HandleWithIdempotency("ledger-worker", func(ctx context.Context, event *events.Event) error {
			slog.Info("ledger-worker: processing event", "type", event.Type, "id", event.ID)
			// TODO: Implement ledger posting logic in Phase 7
			return nil
		}),
	)
	if err != nil {
		slog.Error("failed to subscribe to accounting stream", "error", err)
		os.Exit(1)
	}

	// Company stream: claim verification
	_, err = natsClient.Subscribe(ctx, events.StreamCompany, "claim-worker",
		consumer.HandleWithIdempotency("claim-worker", func(ctx context.Context, event *events.Event) error {
			slog.Info("claim-worker: processing event", "type", event.Type, "id", event.ID)
			// TODO: Implement claim verification logic in Phase 8
			return nil
		}),
	)
	if err != nil {
		slog.Error("failed to subscribe to company stream", "error", err)
		os.Exit(1)
	}

	// Document stream: OCR processing
	_, err = natsClient.Subscribe(ctx, events.StreamDocument, "document-worker",
		consumer.HandleWithIdempotency("document-worker", func(ctx context.Context, event *events.Event) error {
			slog.Info("document-worker: processing event", "type", event.Type, "id", event.ID)
			// TODO: Implement OCR logic in Phase 9
			return nil
		}),
	)
	if err != nil {
		slog.Error("failed to subscribe to document stream", "error", err)
		os.Exit(1)
	}

	// Notification stream: dispatch notifications
	_, err = natsClient.Subscribe(ctx, events.StreamNotification, "notification-worker",
		consumer.HandleWithIdempotency("notification-worker", func(ctx context.Context, event *events.Event) error {
			slog.Info("notification-worker: processing event", "type", event.Type, "id", event.ID)
			// TODO: Implement notification dispatch in Phase 11
			return nil
		}),
	)
	if err != nil {
		slog.Error("failed to subscribe to notification stream", "error", err)
		os.Exit(1)
	}

	// Report stream: report generation
	_, err = natsClient.Subscribe(ctx, events.StreamReport, "report-worker",
		consumer.HandleWithIdempotency("report-worker", func(ctx context.Context, event *events.Event) error {
			slog.Info("report-worker: processing event", "type", event.Type, "id", event.ID)
			// TODO: Implement report generation in Phase 7E
			return nil
		}),
	)
	if err != nil {
		slog.Error("failed to subscribe to report stream", "error", err)
		os.Exit(1)
	}

	slog.Info("all consumers started, waiting for events...")

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down consumer...")
	cancel()
	slog.Info("consumer stopped")
}
