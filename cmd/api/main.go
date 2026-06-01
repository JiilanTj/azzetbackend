package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"codeberg.org/azzet/azzetbe/internal/ai"
	"codeberg.org/azzet/azzetbe/internal/api"
	"codeberg.org/azzet/azzetbe/internal/config"
	"codeberg.org/azzet/azzetbe/internal/database"
	rdb "codeberg.org/azzet/azzetbe/internal/redis"
	"codeberg.org/azzet/azzetbe/internal/shared"
	"codeberg.org/azzet/azzetbe/internal/smtp"

	_ "codeberg.org/azzet/azzetbe/docs"
)

// @title Azzet API
// @version 1.0
// @description Enterprise-grade accounting, tax, and finance platform. User routes use Bearer JWT from /auth/login. Workspace-scoped routes require X-Workspace-ID header. Admin routes under /admin use admin JWT from /admin/auth/login.
// @host localhost:8080
// @BasePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	shared.NewLogger(cfg.AppEnv)
	shared.PrintBanner(cfg.AppPort, cfg.AppEnv)

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

	// SMTP health check (non-fatal — app works without email in dev)
	mailer := smtp.New(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPFrom)
	if err := mailer.HealthCheck(); err != nil {
		slog.Warn("smtp health check failed", "host", cfg.SMTPHost, "error", err)
	} else {
		slog.Info("smtp connected", "host", cfg.SMTPHost)
	}

	// OpenAI health check (non-fatal — AI features degrade gracefully)
	aiClient := ai.NewFromEnv(cfg.OpenAIApiKey, cfg.OpenAIModel)
	aiCtx, aiCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer aiCancel()
	if err := aiClient.HealthCheck(aiCtx); err != nil {
		slog.Warn("openai health check failed", "model", cfg.OpenAIModel, "error", err)
	}

	router := api.NewRouter(cfg, db, redisClient)

	srv := &http.Server{
		Addr:         ":" + cfg.AppPort,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Use error channel so a listen failure is handled in the same select as SIGTERM.
	serverErr := make(chan error, 1)
	go func() {
		slog.Info("starting server", "port", cfg.AppPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		slog.Error("server error", "error", err)
	case <-quit:
		slog.Info("shutting down server...")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("forced shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("server stopped")
}
