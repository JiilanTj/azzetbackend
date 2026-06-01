package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/google/uuid"

	"codeberg.org/azzet/azzetbe/internal/accounting"
	"codeberg.org/azzet/azzetbe/internal/ai"
	"codeberg.org/azzet/azzetbe/internal/claim"
	"codeberg.org/azzet/azzetbe/internal/config"
	"codeberg.org/azzet/azzetbe/internal/database"
	dbpkg "codeberg.org/azzet/azzetbe/internal/db"
	"codeberg.org/azzet/azzetbe/internal/document"
	"codeberg.org/azzet/azzetbe/internal/entity"
	"codeberg.org/azzet/azzetbe/internal/events"
	"codeberg.org/azzet/azzetbe/internal/identity"
	"codeberg.org/azzet/azzetbe/internal/shared"
	"codeberg.org/azzet/azzetbe/internal/storage"
	"codeberg.org/azzet/azzetbe/internal/tax"
	"codeberg.org/azzet/azzetbe/internal/workspace"
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

	// --- Services for workers ---
	queries := dbpkg.New(db.Pool)
	ledgerWorker := accounting.NewLedgerWorker(queries, db.Pool)
	coaService := accounting.NewCOAService(queries, db.Pool)
	itemService := accounting.NewItemService(queries)
	aiClient := ai.NewFromEnv(cfg.OpenAIApiKey, cfg.OpenAIModel)
	categorizer := accounting.NewCategorizer(aiClient)
	accountingService := accounting.NewService(queries, db.Pool, coaService, itemService, categorizer)
	entityService := entity.NewService(queries)
	workspaceService := workspace.NewService(queries, entityService)
	workspaceService.Pool = db.Pool
	identityService := identity.NewService(queries, db.Pool)
	claimWorker := claim.NewClaimWorker(queries, db.Pool, identityService)

	var r2Client *storage.R2Client
	r2Client, _ = storage.NewR2Client(cfg)
	documentService := document.NewService(queries, db.Pool, r2Client, nil)
	docExtractor := document.NewExtractor(aiClient)
	documentWorker := document.NewDocumentWorker(
		queries, db.Pool, r2Client, docExtractor,
		accountingService, identityService, workspaceService, documentService,
	)

	taxService := tax.NewService(queries, db.Pool)
	taxWorker := tax.NewWorker(queries, taxService)

	// --- Register Event Handlers ---

	// User stream: handle user.registered + workspace.created
	_, err = natsClient.Subscribe(ctx, events.StreamUser, "user-entity-worker",
		consumer.HandleWithIdempotency("user-entity-worker", func(ctx context.Context, event *events.Event) error {
			switch event.Type {
			case events.UserRegistered:
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

			case events.WorkspaceCreated:
				var payload struct {
					WorkspaceID string `json:"workspace_id"`
				}
				if err := json.Unmarshal(event.Payload, &payload); err != nil {
					return fmt.Errorf("failed to parse workspace.created payload: %w", err)
				}

				slog.Info("user-entity-worker: workspace created, seeding COA",
					"workspace_id", payload.WorkspaceID,
				)

				wsID, err := parseUUID(payload.WorkspaceID)
				if err != nil {
					return fmt.Errorf("invalid workspace_id: %w", err)
				}

				if err := coaService.SeedDefaultCOA(ctx, wsID); err != nil {
					// If already seeded, that's fine
					if err == accounting.ErrCOAAlreadySeeded {
						slog.Info("user-entity-worker: COA already seeded, skipping",
							"workspace_id", payload.WorkspaceID,
						)
						return nil
					}
					return fmt.Errorf("failed to seed COA: %w", err)
				}

				slog.Info("user-entity-worker: COA seeded successfully",
					"workspace_id", payload.WorkspaceID,
				)
				return nil

			default:
				return nil
			}
		}),
	)
	if err != nil {
		slog.Error("failed to subscribe to user stream", "error", err)
		os.Exit(1)
	}

	// Accounting stream: ledger posting + tax calculation
	_, err = natsClient.Subscribe(ctx, events.StreamAccounting, "ledger-worker",
		consumer.HandleWithIdempotency("ledger-worker", func(ctx context.Context, event *events.Event) error {
			if event.Type == events.TransactionCreated {
				return ledgerWorker.HandleTransactionCreated(ctx, event)
			}
			if event.Type == events.LedgerPosted {
				return taxWorker.HandleLedgerPosted(ctx, event)
			}
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
			switch event.Type {
			case events.CompanyClaimRequested:
				return claimWorker.HandleClaimRequested(ctx, event)
			case events.CompanyClaimApproved:
				return claimWorker.HandleClaimApproved(ctx, event)
			case events.CompanyClaimRejected:
				return nil
			default:
				return nil
			}
		}),
	)
	if err != nil {
		slog.Error("failed to subscribe to company stream", "error", err)
		os.Exit(1)
	}

	// Document stream: OCR processing
	_, err = natsClient.Subscribe(ctx, events.StreamDocument, "document-worker",
		consumer.HandleWithIdempotency("document-worker", func(ctx context.Context, event *events.Event) error {
			switch event.Type {
			case events.DocumentUploaded:
				return documentWorker.HandleDocumentUploaded(ctx, event)
			default:
				return nil
			}
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

	// Report stream: tax report generation
	_, err = natsClient.Subscribe(ctx, events.StreamReport, "report-worker",
		consumer.HandleWithIdempotency("report-worker", func(ctx context.Context, event *events.Event) error {
			if event.Type == events.ReportGenerationReq {
				return taxWorker.HandleReportGeneration(ctx, event)
			}
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

func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}
