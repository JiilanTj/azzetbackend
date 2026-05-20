package subscription_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"codeberg.org/azzet/azzetbe/internal/db"
	"codeberg.org/azzet/azzetbe/internal/subscription"
)

func TestSubscriptionToResponse_Active(t *testing.T) {
	now := time.Now()
	expires := now.Add(30 * 24 * time.Hour)

	sub := &db.TenantSubscription{
		ID:           uuid.New(),
		WorkspaceID:  uuid.New(),
		PlanID:       uuid.New(),
		Status:       subscription.StatusActive,
		BillingCycle: pgtype.Text{String: "monthly", Valid: true},
		StartedAt:    now,
		ExpiresAt:    &expires,
		TrialEndsAt:  nil,
		CancelledAt:  nil,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	resp := subscription.SubscriptionToResponse(sub)

	if resp.ID != sub.ID.String() {
		t.Fatalf("expected ID '%s', got '%s'", sub.ID.String(), resp.ID)
	}
	if resp.WorkspaceID != sub.WorkspaceID.String() {
		t.Fatalf("expected workspace_id '%s', got '%s'", sub.WorkspaceID.String(), resp.WorkspaceID)
	}
	if resp.Status != subscription.StatusActive {
		t.Fatalf("expected status 'active', got '%s'", resp.Status)
	}
	if resp.BillingCycle == nil || *resp.BillingCycle != "monthly" {
		t.Fatalf("expected billing_cycle 'monthly', got %v", resp.BillingCycle)
	}
	if resp.ExpiresAt == nil {
		t.Fatal("expected non-nil expires_at")
	}
	if resp.TrialEndsAt != nil {
		t.Fatalf("expected nil trial_ends_at, got %v", resp.TrialEndsAt)
	}
	if resp.CancelledAt != nil {
		t.Fatalf("expected nil cancelled_at, got %v", resp.CancelledAt)
	}
}

func TestSubscriptionToResponse_Trial(t *testing.T) {
	now := time.Now()
	trialEnd := now.Add(14 * 24 * time.Hour)

	sub := &db.TenantSubscription{
		ID:           uuid.New(),
		WorkspaceID:  uuid.New(),
		PlanID:       uuid.New(),
		Status:       subscription.StatusTrial,
		BillingCycle: pgtype.Text{Valid: false},
		StartedAt:    now,
		ExpiresAt:    &trialEnd,
		TrialEndsAt:  &trialEnd,
		CancelledAt:  nil,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	resp := subscription.SubscriptionToResponse(sub)

	if resp.Status != subscription.StatusTrial {
		t.Fatalf("expected status 'trial', got '%s'", resp.Status)
	}
	if resp.BillingCycle != nil {
		t.Fatalf("expected nil billing_cycle for trial, got %v", resp.BillingCycle)
	}
	if resp.TrialEndsAt == nil {
		t.Fatal("expected non-nil trial_ends_at")
	}
}

func TestSubscriptionToResponse_Cancelled(t *testing.T) {
	now := time.Now()
	cancelledAt := now.Add(-1 * time.Hour)

	sub := &db.TenantSubscription{
		ID:           uuid.New(),
		WorkspaceID:  uuid.New(),
		PlanID:       uuid.New(),
		Status:       subscription.StatusCancelled,
		BillingCycle: pgtype.Text{String: "yearly", Valid: true},
		StartedAt:    now.Add(-30 * 24 * time.Hour),
		ExpiresAt:    nil,
		TrialEndsAt:  nil,
		CancelledAt:  &cancelledAt,
		CreatedAt:    now.Add(-30 * 24 * time.Hour),
		UpdatedAt:    now,
	}

	resp := subscription.SubscriptionToResponse(sub)

	if resp.Status != subscription.StatusCancelled {
		t.Fatalf("expected status 'cancelled', got '%s'", resp.Status)
	}
	if resp.CancelledAt == nil {
		t.Fatal("expected non-nil cancelled_at")
	}
}

func TestSubscriptionToResponse_Free(t *testing.T) {
	now := time.Now()

	sub := &db.TenantSubscription{
		ID:           uuid.New(),
		WorkspaceID:  uuid.New(),
		PlanID:       uuid.New(),
		Status:       subscription.StatusActive,
		BillingCycle: pgtype.Text{Valid: false},
		StartedAt:    now,
		ExpiresAt:    nil, // Free plans don't expire
		TrialEndsAt:  nil,
		CancelledAt:  nil,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	resp := subscription.SubscriptionToResponse(sub)

	if resp.Status != subscription.StatusActive {
		t.Fatalf("expected status 'active', got '%s'", resp.Status)
	}
	if resp.BillingCycle != nil {
		t.Fatalf("expected nil billing_cycle for free plan, got %v", resp.BillingCycle)
	}
	if resp.ExpiresAt != nil {
		t.Fatalf("expected nil expires_at for free plan, got %v", resp.ExpiresAt)
	}
}

func TestSubscriptionWithPlanToResponse(t *testing.T) {
	now := time.Now()
	expires := now.Add(30 * 24 * time.Hour)

	row := &db.GetSubscriptionWithPlanRow{
		ID:           uuid.New(),
		WorkspaceID:  uuid.New(),
		PlanID:       uuid.New(),
		Status:       subscription.StatusActive,
		BillingCycle: pgtype.Text{String: "monthly", Valid: true},
		StartedAt:    now,
		ExpiresAt:    &expires,
		TrialEndsAt:  nil,
		CancelledAt:  nil,
		CreatedAt:    now,
		UpdatedAt:    now,
		PlanName:     "Professional",
		PlanSlug:     "professional",
		PlanType:     "paid",
		PlanTier:     2,
	}

	resp := subscription.SubscriptionWithPlanToResponse(row)

	if resp.PlanName == nil || *resp.PlanName != "Professional" {
		t.Fatalf("expected plan_name 'Professional', got %v", resp.PlanName)
	}
	if resp.PlanSlug == nil || *resp.PlanSlug != "professional" {
		t.Fatalf("expected plan_slug 'professional', got %v", resp.PlanSlug)
	}
}

func TestSubscriptionConstants(t *testing.T) {
	if subscription.StatusActive != "active" {
		t.Fatalf("expected 'active', got '%s'", subscription.StatusActive)
	}
	if subscription.StatusTrial != "trial" {
		t.Fatalf("expected 'trial', got '%s'", subscription.StatusTrial)
	}
	if subscription.StatusExpired != "expired" {
		t.Fatalf("expected 'expired', got '%s'", subscription.StatusExpired)
	}
	if subscription.StatusCancelled != "cancelled" {
		t.Fatalf("expected 'cancelled', got '%s'", subscription.StatusCancelled)
	}
	if subscription.CycleMonthly != "monthly" {
		t.Fatalf("expected 'monthly', got '%s'", subscription.CycleMonthly)
	}
	if subscription.CycleYearly != "yearly" {
		t.Fatalf("expected 'yearly', got '%s'", subscription.CycleYearly)
	}
}
