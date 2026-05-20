package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/hibiken/asynq"

	"codeberg.org/azzet/azzetbe/internal/config"
	"codeberg.org/azzet/azzetbe/internal/database"
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
	fmt.Printf("  %s╭─── Azzet Worker ────────────────────────────────────────╮%s\n", "\033[36m", "\033[0m")
	fmt.Printf("  %s│                                                         │%s\n", "\033[36m", "\033[0m")
	fmt.Printf("  %s│   %sBackground Task Worker%s                               │%s\n", "\033[36m", "\033[1m\033[37m", "\033[0m", "\033[36m")
	fmt.Printf("  %s│                                                         │%s\n", "\033[36m", "\033[0m")
	fmt.Printf("  %s│   %s●%s Redis       %s→%s %s%-35s%s│%s\n", "\033[36m", "\033[32m", "\033[0m", "\033[1m", "\033[0m", "\033[37m", cfg.RedisAddr(), "\033[36m", "\033[0m")
	fmt.Printf("  %s│   %s●%s Concurrency %s→%s %s%-35d%s│%s\n", "\033[36m", "\033[32m", "\033[0m", "\033[1m", "\033[0m", "\033[33m", cfg.WorkerConcurrency, "\033[36m", "\033[0m")
	fmt.Printf("  %s│   %s●%s Env         %s→%s %s%-35s%s│%s\n", "\033[36m", "\033[32m", "\033[0m", "\033[1m", "\033[0m", "\033[33m", cfg.AppEnv, "\033[36m", "\033[0m")
	fmt.Printf("  %s│                                                         │%s\n", "\033[36m", "\033[0m")
	fmt.Printf("  %s╰─────────────────────────────────────────────────────────╯%s\n", "\033[36m", "\033[0m")
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

	// Use parsed Redis address for asynq (host:port format)
	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: cfg.RedisAddr()},
		asynq.Config{
			Concurrency: cfg.WorkerConcurrency,
			Logger:      nil,
		},
	)

	mux := asynq.NewServeMux()

	// TODO: Register task handlers here
	// mux.HandleFunc("email:send", handlers.HandleEmailSend)
	// mux.HandleFunc("image:process", handlers.HandleImageProcess)
	// mux.HandleFunc("webhook:retry", handlers.HandleWebhookRetry)

	slog.Info("worker started", "concurrency", cfg.WorkerConcurrency)

	// Use error channel instead of os.Exit in goroutine
	workerErr := make(chan error, 1)
	go func() {
		if err := srv.Run(mux); err != nil {
			workerErr <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-workerErr:
		slog.Error("worker error", "error", err)
	case <-quit:
		slog.Info("shutting down worker...")
	}

	srv.Shutdown()
	slog.Info("worker stopped")
}
