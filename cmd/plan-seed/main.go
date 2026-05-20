package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"codeberg.org/azzet/azzetbe/internal/config"
	"codeberg.org/azzet/azzetbe/internal/database"
	"codeberg.org/azzet/azzetbe/internal/db"
	"codeberg.org/azzet/azzetbe/internal/shared"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	shared.NewLogger(cfg.AppEnv)

	database, err := database.NewFromEnv(cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	queries := db.New(database.Pool)
	ctx := context.Background()

	fmt.Println()
	fmt.Println("  Seeding plans...")
	fmt.Println()

	plans := []seedPlan{
		{
			Name:         "Free",
			Slug:         "free",
			Description:  "Get started with basic features. Perfect for personal use.",
			Type:         "free",
			PriceMonthly: 0,
			PriceYearly:  0,
			IsTrial:      false,
			TrialDays:    0,
			Tier:         0,
			Features: []seedFeature{
				{Key: "max_entities", Type: "quota", Int: intPtr(5)},
				{Key: "max_transactions_monthly", Type: "quota", Int: intPtr(100)},
				{Key: "max_users", Type: "quota", Int: intPtr(1)},
				{Key: "ocr_enabled", Type: "boolean", Bool: boolPtr(false)},
				{Key: "ocr_requests_monthly", Type: "quota", Int: intPtr(0)},
				{Key: "export_pdf", Type: "boolean", Bool: boolPtr(false)},
				{Key: "export_excel", Type: "boolean", Bool: boolPtr(false)},
				{Key: "api_access", Type: "boolean", Bool: boolPtr(false)},
				{Key: "webhook_enabled", Type: "boolean", Bool: boolPtr(false)},
				{Key: "multi_user", Type: "boolean", Bool: boolPtr(false)},
				{Key: "custom_roles", Type: "boolean", Bool: boolPtr(false)},
				{Key: "audit_logs", Type: "boolean", Bool: boolPtr(false)},
				{Key: "priority_support", Type: "boolean", Bool: boolPtr(false)},
				{Key: "white_label", Type: "boolean", Bool: boolPtr(false)},
				{Key: "report_financial", Type: "boolean", Bool: boolPtr(true)},
				{Key: "report_tax", Type: "boolean", Bool: boolPtr(false)},
			},
		},
		{
			Name:         "Starter",
			Slug:         "starter",
			Description:  "For small businesses getting organized. Includes OCR and multi-user.",
			Type:         "paid",
			PriceMonthly: 99000,
			PriceYearly:  990000,
			IsTrial:      true,
			TrialDays:    14,
			Tier:         1,
			Features: []seedFeature{
				{Key: "max_entities", Type: "quota", Int: intPtr(50)},
				{Key: "max_transactions_monthly", Type: "quota", Int: intPtr(1000)},
				{Key: "max_users", Type: "quota", Int: intPtr(5)},
				{Key: "ocr_enabled", Type: "boolean", Bool: boolPtr(true)},
				{Key: "ocr_requests_monthly", Type: "quota", Int: intPtr(50)},
				{Key: "export_pdf", Type: "boolean", Bool: boolPtr(true)},
				{Key: "export_excel", Type: "boolean", Bool: boolPtr(false)},
				{Key: "api_access", Type: "boolean", Bool: boolPtr(false)},
				{Key: "webhook_enabled", Type: "boolean", Bool: boolPtr(false)},
				{Key: "multi_user", Type: "boolean", Bool: boolPtr(true)},
				{Key: "custom_roles", Type: "boolean", Bool: boolPtr(false)},
				{Key: "audit_logs", Type: "boolean", Bool: boolPtr(false)},
				{Key: "priority_support", Type: "boolean", Bool: boolPtr(false)},
				{Key: "white_label", Type: "boolean", Bool: boolPtr(false)},
				{Key: "report_financial", Type: "boolean", Bool: boolPtr(true)},
				{Key: "report_tax", Type: "boolean", Bool: boolPtr(true)},
			},
		},
		{
			Name:         "Professional",
			Slug:         "professional",
			Description:  "For growing businesses. Full accounting, API access, and advanced features.",
			Type:         "paid",
			PriceMonthly: 299000,
			PriceYearly:  2990000,
			IsTrial:      true,
			TrialDays:    14,
			Tier:         2,
			Features: []seedFeature{
				{Key: "max_entities", Type: "quota", Int: intPtr(500)},
				{Key: "max_transactions_monthly", Type: "quota", Int: intPtr(10000)},
				{Key: "max_users", Type: "quota", Int: intPtr(25)},
				{Key: "ocr_enabled", Type: "boolean", Bool: boolPtr(true)},
				{Key: "ocr_requests_monthly", Type: "quota", Int: intPtr(500)},
				{Key: "export_pdf", Type: "boolean", Bool: boolPtr(true)},
				{Key: "export_excel", Type: "boolean", Bool: boolPtr(true)},
				{Key: "api_access", Type: "boolean", Bool: boolPtr(true)},
				{Key: "webhook_enabled", Type: "boolean", Bool: boolPtr(true)},
				{Key: "multi_user", Type: "boolean", Bool: boolPtr(true)},
				{Key: "custom_roles", Type: "boolean", Bool: boolPtr(true)},
				{Key: "audit_logs", Type: "boolean", Bool: boolPtr(true)},
				{Key: "priority_support", Type: "boolean", Bool: boolPtr(false)},
				{Key: "white_label", Type: "boolean", Bool: boolPtr(false)},
				{Key: "report_financial", Type: "boolean", Bool: boolPtr(true)},
				{Key: "report_tax", Type: "boolean", Bool: boolPtr(true)},
			},
		},
		{
			Name:         "Enterprise",
			Slug:         "enterprise",
			Description:  "For large organizations. Unlimited everything, white-label, and priority support.",
			Type:         "paid",
			PriceMonthly: 799000,
			PriceYearly:  7990000,
			IsTrial:      true,
			TrialDays:    30,
			Tier:         3,
			Features: []seedFeature{
				{Key: "max_entities", Type: "quota", Int: intPtr(-1)},
				{Key: "max_transactions_monthly", Type: "quota", Int: intPtr(-1)},
				{Key: "max_users", Type: "quota", Int: intPtr(-1)},
				{Key: "ocr_enabled", Type: "boolean", Bool: boolPtr(true)},
				{Key: "ocr_requests_monthly", Type: "quota", Int: intPtr(-1)},
				{Key: "export_pdf", Type: "boolean", Bool: boolPtr(true)},
				{Key: "export_excel", Type: "boolean", Bool: boolPtr(true)},
				{Key: "api_access", Type: "boolean", Bool: boolPtr(true)},
				{Key: "webhook_enabled", Type: "boolean", Bool: boolPtr(true)},
				{Key: "multi_user", Type: "boolean", Bool: boolPtr(true)},
				{Key: "custom_roles", Type: "boolean", Bool: boolPtr(true)},
				{Key: "audit_logs", Type: "boolean", Bool: boolPtr(true)},
				{Key: "priority_support", Type: "boolean", Bool: boolPtr(true)},
				{Key: "white_label", Type: "boolean", Bool: boolPtr(true)},
				{Key: "report_financial", Type: "boolean", Bool: boolPtr(true)},
				{Key: "report_tax", Type: "boolean", Bool: boolPtr(true)},
			},
		},
	}

	for _, p := range plans {
		// Check if plan already exists
		exists, err := queries.ExistsPlanBySlug(ctx, p.Slug)
		if err != nil {
			slog.Error("failed to check plan", "slug", p.Slug, "error", err)
			continue
		}
		if exists {
			fmt.Printf("  [skip] Plan '%s' already exists\n", p.Name)
			continue
		}

		now := time.Now()
		plan, err := queries.CreatePlan(ctx, db.CreatePlanParams{
			ID:           uuid.New(),
			Name:         p.Name,
			Slug:         p.Slug,
			Description:  pgtype.Text{String: p.Description, Valid: true},
			Type:         p.Type,
			PriceMonthly: numericFromFloat(p.PriceMonthly),
			PriceYearly:  numericFromFloat(p.PriceYearly),
			IsTrial:      p.IsTrial,
			TrialDays:    int32(p.TrialDays),
			Tier:         int32(p.Tier),
			IsActive:     true,
			CreatedAt:    now,
			UpdatedAt:    now,
		})
		if err != nil {
			slog.Error("failed to create plan", "name", p.Name, "error", err)
			continue
		}

		fmt.Printf("  [created] Plan '%s' (tier %d, %s)\n", p.Name, p.Tier, p.Type)

		// Seed features
		for _, f := range p.Features {
			var valueBool pgtype.Bool
			var valueInt pgtype.Int4
			var valueText pgtype.Text

			switch f.Type {
			case "boolean":
				if f.Bool != nil {
					valueBool = pgtype.Bool{Bool: *f.Bool, Valid: true}
				}
			case "quota":
				if f.Int != nil {
					valueInt = pgtype.Int4{Int32: int32(*f.Int), Valid: true}
				}
			case "tier":
				if f.Text != nil {
					valueText = pgtype.Text{String: *f.Text, Valid: true}
				}
			}

			_, err := queries.UpsertPlanFeature(ctx, db.UpsertPlanFeatureParams{
				ID:          uuid.New(),
				PlanID:      plan.ID,
				FeatureKey:  f.Key,
				FeatureType: f.Type,
				ValueBool:   valueBool,
				ValueInt:    valueInt,
				ValueText:   valueText,
				CreatedAt:   now,
			})
			if err != nil {
				slog.Error("failed to create feature", "plan", p.Name, "feature", f.Key, "error", err)
				continue
			}
		}

		fmt.Printf("           → %d features added\n", len(p.Features))
	}

	fmt.Println()
	fmt.Println("  ✓ Plan seeding complete!")
	fmt.Println()
}

// --- Types ---

type seedPlan struct {
	Name         string
	Slug         string
	Description  string
	Type         string
	PriceMonthly float64
	PriceYearly  float64
	IsTrial      bool
	TrialDays    int
	Tier         int
	Features     []seedFeature
}

type seedFeature struct {
	Key  string
	Type string
	Bool *bool
	Int  *int
	Text *string
}

// --- Helpers ---

func boolPtr(b bool) *bool    { return &b }
func intPtr(i int) *int       { return &i }
func strPtr(s string) *string { return &s }

func numericFromFloat(f float64) pgtype.Numeric {
	var n pgtype.Numeric
	n.Scan(fmt.Sprintf("%.2f", f))
	return n
}
