package main

import (
	"context"
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
	fmt.Printf("  \033[36m╭─── Azzet Publisher ─────────────────────────────────────╮\033[0m\n")
	fmt.Printf("  \033[36m│                                                         │\033[0m\n")
	fmt.Printf("  \033[36m│   \033[1m\033[37mOutbox → NATS JetStream Publisher\033[0m\033[36m                    │\033[0m\n")
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

	// Ensure all streams exist
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := natsClient.EnsureStreams(ctx); err != nil {
		slog.Error("failed to ensure nats streams", "error", err)
		os.Exit(1)
	}

	// Start publisher
	publisher := events.NewPublisher(db.Pool, natsClient)

	publisherErr := make(chan error, 1)
	go func() {
		if err := publisher.Run(ctx); err != nil {
			publisherErr <- err
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-publisherErr:
		slog.Error("publisher error", "error", err)
	case <-quit:
		slog.Info("shutting down publisher...")
	}

	cancel()
	slog.Info("publisher stopped")
}
