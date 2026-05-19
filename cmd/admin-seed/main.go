package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"

	"codeberg.org/azzet/azzetbe/internal/config"
	"codeberg.org/azzet/azzetbe/internal/database"
	"codeberg.org/azzet/azzetbe/internal/shared"
)

func main() {
	email := flag.String("email", "", "Admin email (required)")
	password := flag.String("password", "", "Admin password (min 12 chars, required)")
	name := flag.String("name", "Super Admin", "Admin display name")
	flag.Parse()

	if *email == "" || *password == "" {
		fmt.Println("Usage: admin-seed --email=admin@azzet.com --password=YourSecurePass123!")
		fmt.Println()
		fmt.Println("Flags:")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if len(*password) < 12 {
		fmt.Println("Error: password must be at least 12 characters")
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	db, err := database.NewFromEnv(cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Check if admin already exists
	var exists bool
	err = db.Pool.QueryRow(context.Background(), `SELECT EXISTS(SELECT 1 FROM platform_admins WHERE email = $1)`, *email).Scan(&exists)
	if err != nil {
		slog.Error("failed to check existing admin", "error", err)
		os.Exit(1)
	}
	if exists {
		fmt.Printf("Admin with email '%s' already exists\n", *email)
		os.Exit(1)
	}

	// Hash password
	hash, err := shared.HashPassword(*password)
	if err != nil {
		slog.Error("failed to hash password", "error", err)
		os.Exit(1)
	}

	// Create admin
	id := uuid.New()
	now := time.Now()
	_, err = db.Pool.Exec(context.Background(), `
		INSERT INTO platform_admins (id, email, password_hash, name, role, mfa_enabled, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'SUPER_ADMIN', FALSE, 'ACTIVE', $5, $6)
	`, id, *email, hash, *name, now, now)
	if err != nil {
		slog.Error("failed to create admin", "error", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("  ✓ Super Admin created successfully!")
	fmt.Println()
	fmt.Printf("  ID:    %s\n", id.String())
	fmt.Printf("  Email: %s\n", *email)
	fmt.Printf("  Name:  %s\n", *name)
	fmt.Printf("  Role:  SUPER_ADMIN\n")
	fmt.Println()
	fmt.Println("  ⚠ MFA is not enabled yet. Login to set up MFA.")
	fmt.Println()
}
