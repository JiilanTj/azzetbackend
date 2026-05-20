package subscription

import (
	"time"

	"codeberg.org/azzet/azzetbe/internal/db"
)

// --- Request DTOs ---

// SubscribeRequest represents subscribing to a plan
// @Description Subscribe a workspace to a plan
type SubscribeRequest struct {
	PlanID       string  `json:"plan_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	BillingCycle *string `json:"billing_cycle,omitempty" example:"monthly" enums:"monthly,yearly"`
}

// --- Response DTOs ---

// SubscriptionResponse represents an active subscription
// @Description Workspace subscription information
type SubscriptionResponse struct {
	ID           string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	WorkspaceID  string  `json:"workspace_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	PlanID       string  `json:"plan_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	PlanName     *string `json:"plan_name,omitempty" example:"Professional"`
	PlanSlug     *string `json:"plan_slug,omitempty" example:"professional"`
	Status       string  `json:"status" example:"active" enums:"active,trial,expired,cancelled"`
	BillingCycle *string `json:"billing_cycle,omitempty" example:"monthly"`
	StartedAt    string  `json:"started_at" example:"2026-05-20T10:00:00Z"`
	ExpiresAt    *string `json:"expires_at,omitempty" example:"2026-06-20T10:00:00Z"`
	TrialEndsAt  *string `json:"trial_ends_at,omitempty" example:"2026-06-03T10:00:00Z"`
	CancelledAt  *string `json:"cancelled_at,omitempty"`
	CreatedAt    string  `json:"created_at" example:"2026-05-20T10:00:00Z"`
}

// UsageResponse represents quota usage for a feature
// @Description Feature usage tracking
type UsageResponse struct {
	FeatureKey  string `json:"feature_key" example:"max_transactions_monthly"`
	UsageCount  int    `json:"usage_count" example:"42"`
	Limit       int    `json:"limit" example:"1000"`
	PeriodStart string `json:"period_start" example:"2026-05-01T00:00:00Z"`
	PeriodEnd   string `json:"period_end" example:"2026-06-01T00:00:00Z"`
}

// MessageResponse represents a simple message
// @Description Simple message response
type MessageResponse struct {
	Message string `json:"message" example:"Operation successful"`
}

// --- Constants ---

const (
	StatusActive    = "active"
	StatusTrial     = "trial"
	StatusExpired   = "expired"
	StatusCancelled = "cancelled"

	CycleMonthly = "monthly"
	CycleYearly  = "yearly"
)

// --- Converters ---

func SubscriptionToResponse(s *db.TenantSubscription) SubscriptionResponse {
	resp := SubscriptionResponse{
		ID:          s.ID.String(),
		WorkspaceID: s.WorkspaceID.String(),
		PlanID:      s.PlanID.String(),
		Status:      s.Status,
		StartedAt:   s.StartedAt.Format(time.RFC3339),
		CreatedAt:   s.CreatedAt.Format(time.RFC3339),
	}

	if s.BillingCycle.Valid {
		resp.BillingCycle = &s.BillingCycle.String
	}
	if s.ExpiresAt != nil {
		t := s.ExpiresAt.Format(time.RFC3339)
		resp.ExpiresAt = &t
	}
	if s.TrialEndsAt != nil {
		t := s.TrialEndsAt.Format(time.RFC3339)
		resp.TrialEndsAt = &t
	}
	if s.CancelledAt != nil {
		t := s.CancelledAt.Format(time.RFC3339)
		resp.CancelledAt = &t
	}

	return resp
}

func SubscriptionWithPlanToResponse(s *db.GetSubscriptionWithPlanRow) SubscriptionResponse {
	resp := SubscriptionResponse{
		ID:          s.ID.String(),
		WorkspaceID: s.WorkspaceID.String(),
		PlanID:      s.PlanID.String(),
		PlanName:    &s.PlanName,
		PlanSlug:    &s.PlanSlug,
		Status:      s.Status,
		StartedAt:   s.StartedAt.Format(time.RFC3339),
		CreatedAt:   s.CreatedAt.Format(time.RFC3339),
	}

	if s.BillingCycle.Valid {
		resp.BillingCycle = &s.BillingCycle.String
	}
	if s.ExpiresAt != nil {
		t := s.ExpiresAt.Format(time.RFC3339)
		resp.ExpiresAt = &t
	}
	if s.TrialEndsAt != nil {
		t := s.TrialEndsAt.Format(time.RFC3339)
		resp.TrialEndsAt = &t
	}
	if s.CancelledAt != nil {
		t := s.CancelledAt.Format(time.RFC3339)
		resp.CancelledAt = &t
	}

	return resp
}
