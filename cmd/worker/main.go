package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/hibiken/asynq"

	"codeberg.org/azzet/azzetbe/internal/config"
	"codeberg.org/azzet/azzetbe/internal/database"
	"codeberg.org/azzet/azzetbe/internal/events"
	rdb "codeberg.org/azzet/azzetbe/internal/redis"
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
	fmt.Printf("  \033[36m╭─── Azzet Worker ────────────────────────────────────────╮\033[0m\n")
	fmt.Printf("  \033[36m│                                                         │\033[0m\n")
	fmt.Printf("  \033[36m│   \033[1m\033[37mBackground Task Worker (Asynq)\033[0m\033[36m                       │\033[0m\n")
	fmt.Printf("  \033[36m│                                                         │\033[0m\n")
	fmt.Printf("  \033[36m│   \033[32m●\033[0m Redis       \033[1m→\033[0m \033[37m%-35s\033[36m│\033[0m\n", cfg.RedisAddr())
	fmt.Printf("  \033[36m│   \033[32m●\033[0m Concurrency \033[1m→\033[0m \033[33m%-35d\033[36m│\033[0m\n", cfg.WorkerConcurrency)
	fmt.Printf("  \033[36m│   \033[32m●\033[0m Env         \033[1m→\033[0m \033[33m%-35s\033[36m│\033[0m\n", cfg.AppEnv)
	fmt.Printf("  \033[36m│                                                         │\033[0m\n")
	fmt.Printf("  \033[36m╰─────────────────────────────────────────────────────────╯\033[0m\n")
	fmt.Println()

	db, err := database.NewFromEnv(cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	redisClient, err := rdb.NewFromURL(cfg.RedisURL)
	if err != nil {
		slog.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()

	// Asynq server
	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: cfg.RedisAddr()},
		asynq.Config{
			Concurrency: cfg.WorkerConcurrency,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
			Logger: nil,
		},
	)

	mux := asynq.NewServeMux()

	// --- Register Task Handlers ---

	// Email tasks
	mux.HandleFunc(events.TaskEmailSend, handleEmailSend)
	mux.HandleFunc(events.TaskEmailVerification, handleEmailVerification)
	mux.HandleFunc(events.TaskEmailPasswordReset, handleEmailPasswordReset)
	mux.HandleFunc(events.TaskEmailInvoice, handleEmailInvoice)

	// Image tasks
	mux.HandleFunc(events.TaskImageResize, handleImageResize)
	mux.HandleFunc(events.TaskImageOCR, handleImageOCR)

	// Webhook tasks
	mux.HandleFunc(events.TaskWebhookDeliver, handleWebhookDeliver)
	mux.HandleFunc(events.TaskWebhookRetry, handleWebhookRetry)

	// Invoice tasks
	mux.HandleFunc(events.TaskInvoiceGenerate, handleInvoiceGenerate)
	mux.HandleFunc(events.TaskInvoiceReminder, handleInvoiceReminder)

	// Report tasks
	mux.HandleFunc(events.TaskReportGenerate, handleReportGenerate)

	// Cleanup tasks
	mux.HandleFunc(events.TaskCleanupSessions, handleCleanupSessions)
	mux.HandleFunc(events.TaskCleanupTokens, handleCleanupTokens)
	mux.HandleFunc(events.TaskCleanupOutbox, handleCleanupOutbox)
	mux.HandleFunc(events.TaskSubscriptionCheck, handleSubscriptionCheck)

	slog.Info("worker started", "concurrency", cfg.WorkerConcurrency)

	// Start worker
	workerErr := make(chan error, 1)
	go func() {
		if err := srv.Run(mux); err != nil {
			workerErr <- err
		}
	}()

	// Setup scheduled tasks (cron)
	scheduler := asynq.NewScheduler(
		asynq.RedisClientOpt{Addr: cfg.RedisAddr()},
		nil,
	)

	// Cleanup expired sessions daily at 3 AM
	scheduler.Register("0 3 * * *", asynq.NewTask(events.TaskCleanupSessions, nil))
	// Cleanup expired tokens daily at 3:30 AM
	scheduler.Register("30 3 * * *", asynq.NewTask(events.TaskCleanupTokens, nil))
	// Cleanup old outbox events daily at 4 AM
	scheduler.Register("0 4 * * *", asynq.NewTask(events.TaskCleanupOutbox, nil))
	// Check subscription expiry every hour
	scheduler.Register("0 * * * *", asynq.NewTask(events.TaskSubscriptionCheck, nil))

	go func() {
		if err := scheduler.Run(); err != nil {
			slog.Error("scheduler error", "error", err)
		}
	}()

	// Wait for shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-workerErr:
		slog.Error("worker error", "error", err)
	case <-quit:
		slog.Info("shutting down worker...")
	}

	srv.Shutdown()
	scheduler.Shutdown()
	slog.Info("worker stopped")
}

// --- Task Handlers ---

func handleEmailSend(ctx context.Context, task *asynq.Task) error {
	var payload map[string]string
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("email:send: invalid payload: %w", err)
	}
	slog.Info("email:send", "to", payload["to"], "subject", payload["subject"])
	// TODO: Implement actual email sending via SMTP
	return nil
}

func handleEmailVerification(ctx context.Context, task *asynq.Task) error {
	slog.Info("email:verification processing")
	// TODO: Send verification email
	return nil
}

func handleEmailPasswordReset(ctx context.Context, task *asynq.Task) error {
	slog.Info("email:password_reset processing")
	// TODO: Send password reset email
	return nil
}

func handleEmailInvoice(ctx context.Context, task *asynq.Task) error {
	slog.Info("email:invoice processing")
	// TODO: Send invoice email
	return nil
}

func handleImageResize(ctx context.Context, task *asynq.Task) error {
	slog.Info("image:resize processing")
	// TODO: Resize image and upload to R2
	return nil
}

func handleImageOCR(ctx context.Context, task *asynq.Task) error {
	slog.Info("image:ocr processing")
	// TODO: OCR via OpenAI and store results
	return nil
}

func handleWebhookDeliver(ctx context.Context, task *asynq.Task) error {
	slog.Info("webhook:deliver processing")
	// TODO: Deliver webhook to external URL
	return nil
}

func handleWebhookRetry(ctx context.Context, task *asynq.Task) error {
	slog.Info("webhook:retry processing")
	// TODO: Retry failed webhook delivery
	return nil
}

func handleInvoiceGenerate(ctx context.Context, task *asynq.Task) error {
	slog.Info("invoice:generate processing")
	// TODO: Generate invoice PDF
	return nil
}

func handleInvoiceReminder(ctx context.Context, task *asynq.Task) error {
	slog.Info("invoice:reminder processing")
	// TODO: Send invoice payment reminder
	return nil
}

func handleReportGenerate(ctx context.Context, task *asynq.Task) error {
	slog.Info("report:generate processing")
	// TODO: Generate financial report
	return nil
}

func handleCleanupSessions(ctx context.Context, task *asynq.Task) error {
	slog.Info("cleanup:sessions processing")
	// TODO: Delete expired sessions from database
	return nil
}

func handleCleanupTokens(ctx context.Context, task *asynq.Task) error {
	slog.Info("cleanup:tokens processing")
	// TODO: Delete expired OTPs and blacklisted tokens
	return nil
}

func handleCleanupOutbox(ctx context.Context, task *asynq.Task) error {
	slog.Info("cleanup:outbox processing")
	// TODO: Delete published outbox events older than 14 days
	return nil
}

func handleSubscriptionCheck(ctx context.Context, task *asynq.Task) error {
	slog.Info("subscription:check_expiry processing")
	// TODO: Expire trial subscriptions that have passed trial_ends_at
	return nil
}
