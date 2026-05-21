package subscription

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"codeberg.org/azzet/azzetbe/internal/billing"
	"codeberg.org/azzet/azzetbe/internal/db"
)

var ErrNoActiveSubscription = errors.New("no active subscription")
var ErrAlreadySubscribed = errors.New("workspace already has an active subscription")
var ErrFeatureNotAvailable = errors.New("feature not available in current plan")
var ErrQuotaExceeded = errors.New("quota exceeded for current plan")

type Service struct {
	Queries        *db.Queries
	BillingService *billing.Service
}

func NewService(queries *db.Queries) *Service {
	return &Service{Queries: queries}
}

// Subscribe subscribes a workspace to a plan
func (s *Service) Subscribe(ctx context.Context, workspaceID string, req *SubscribeRequest) (*SubscriptionResponse, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("invalid workspace_id")
	}

	planID, err := uuid.Parse(req.PlanID)
	if err != nil {
		return nil, fmt.Errorf("invalid plan_id")
	}

	// Check if already has active subscription
	_, err = s.Queries.GetActiveSubscription(ctx, wsID)
	if err == nil {
		return nil, ErrAlreadySubscribed
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	// Get plan details
	plan, err := s.Queries.GetPlanByID(ctx, planID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("plan not found")
		}
		return nil, err
	}

	if !plan.IsActive {
		return nil, fmt.Errorf("plan is not available")
	}

	now := time.Now()
	params := db.CreateSubscriptionParams{
		ID:          uuid.New(),
		WorkspaceID: wsID,
		PlanID:      planID,
		StartedAt:   now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Determine subscription type
	if plan.Type == "free" {
		params.Status = StatusActive
		// Free plans don't expire
	} else if plan.IsTrial && req.BillingCycle == nil {
		// Start trial
		params.Status = StatusTrial
		trialEnd := now.Add(time.Duration(plan.TrialDays) * 24 * time.Hour)
		params.TrialEndsAt = &trialEnd
		params.ExpiresAt = &trialEnd
	} else {
		// Paid plan — create with pending_payment, then generate invoice + payment
		params.Status = StatusPendingPayment
		if req.BillingCycle != nil {
			params.BillingCycle = pgtype.Text{String: *req.BillingCycle, Valid: true}
			switch *req.BillingCycle {
			case CycleMonthly:
				exp := now.Add(30 * 24 * time.Hour)
				params.ExpiresAt = &exp
			case CycleYearly:
				exp := now.Add(365 * 24 * time.Hour)
				params.ExpiresAt = &exp
			}
		}
	}

	sub, err := s.Queries.CreateSubscription(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	resp := SubscriptionToResponse(&sub)

	// For paid plans, create invoice and initiate payment via Xendit
	if plan.Type != "free" && !(plan.IsTrial && req.BillingCycle == nil) {
		if s.BillingService == nil {
			return nil, fmt.Errorf("billing service not configured")
		}

		// Determine amount based on billing cycle
		var amount float64
		cycle := "monthly"
		if req.BillingCycle != nil {
			cycle = *req.BillingCycle
		}
		if cycle == CycleYearly {
			amount = numericToFloat(plan.PriceYearly)
		} else {
			amount = numericToFloat(plan.PriceMonthly)
		}

		description := fmt.Sprintf("%s - %s", plan.Name, cycle)

		// Create invoice
		invoice, err := s.BillingService.CreateInvoice(ctx, wsID.String(), sub.ID.String(), amount, description)
		if err != nil {
			return nil, fmt.Errorf("failed to create invoice: %w", err)
		}

		// Initiate payment via Xendit
		payment, err := s.BillingService.PayInvoice(ctx, wsID.String(), invoice.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to initiate payment: %w", err)
		}

		resp.PaymentURL = payment.PaymentURL
	}

	return &resp, nil
}

// StartTrial starts a trial for a workspace on a specific plan
func (s *Service) StartTrial(ctx context.Context, workspaceID, planID string) (*SubscriptionResponse, error) {
	return s.Subscribe(ctx, workspaceID, &SubscribeRequest{PlanID: planID})
}

// GetActiveSubscription returns the active subscription for a workspace
func (s *Service) GetActiveSubscription(ctx context.Context, workspaceID string) (*SubscriptionResponse, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("invalid workspace_id")
	}

	sub, err := s.Queries.GetSubscriptionWithPlan(ctx, wsID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoActiveSubscription
		}
		return nil, err
	}

	// Check if trial expired
	if sub.Status == StatusTrial && sub.TrialEndsAt != nil && time.Now().After(*sub.TrialEndsAt) {
		_ = s.Queries.ExpireSubscription(ctx, sub.ID)
		return nil, ErrNoActiveSubscription
	}

	// Check if subscription expired
	if sub.ExpiresAt != nil && time.Now().After(*sub.ExpiresAt) {
		_ = s.Queries.ExpireSubscription(ctx, sub.ID)
		return nil, ErrNoActiveSubscription
	}

	resp := SubscriptionWithPlanToResponse(&sub)
	return &resp, nil
}

// ListSubscriptions returns all subscriptions for a workspace
func (s *Service) ListSubscriptions(ctx context.Context, workspaceID string) ([]SubscriptionResponse, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("invalid workspace_id")
	}

	subs, err := s.Queries.ListSubscriptionsByWorkspace(ctx, wsID)
	if err != nil {
		return nil, err
	}

	var resp []SubscriptionResponse
	for i := range subs {
		resp = append(resp, SubscriptionToResponse(&subs[i]))
	}
	if resp == nil {
		resp = []SubscriptionResponse{}
	}
	return resp, nil
}

// Cancel cancels the active subscription
func (s *Service) Cancel(ctx context.Context, workspaceID string) error {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return fmt.Errorf("invalid workspace_id")
	}

	sub, err := s.Queries.GetActiveSubscription(ctx, wsID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNoActiveSubscription
		}
		return err
	}

	return s.Queries.CancelSubscription(ctx, sub.ID)
}

// ChangePlan upgrades or downgrades the plan
func (s *Service) ChangePlan(ctx context.Context, workspaceID string, req *SubscribeRequest) (*SubscriptionResponse, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("invalid workspace_id")
	}

	// Cancel current subscription
	currentSub, err := s.Queries.GetActiveSubscription(ctx, wsID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if err == nil {
		_ = s.Queries.CancelSubscription(ctx, currentSub.ID)
	}

	// Create new subscription
	return s.Subscribe(ctx, workspaceID, req)
}

// HasFeature checks if the workspace's current plan has a boolean feature enabled
func (s *Service) HasFeature(ctx context.Context, workspaceID, featureKey string) (bool, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return false, fmt.Errorf("invalid workspace_id")
	}

	sub, err := s.Queries.GetActiveSubscription(ctx, wsID)
	if err != nil {
		return false, ErrNoActiveSubscription
	}

	feature, err := s.Queries.GetPlanFeatureByKey(ctx, db.GetPlanFeatureByKeyParams{
		PlanID:     sub.PlanID,
		FeatureKey: featureKey,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil // Feature not defined = not available
		}
		return false, err
	}

	if feature.FeatureType == "boolean" && feature.ValueBool.Valid {
		return feature.ValueBool.Bool, nil
	}

	return false, nil
}

// CheckQuota checks if the workspace has remaining quota for a feature
func (s *Service) CheckQuota(ctx context.Context, workspaceID, featureKey string) (bool, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return false, fmt.Errorf("invalid workspace_id")
	}

	sub, err := s.Queries.GetActiveSubscription(ctx, wsID)
	if err != nil {
		return false, ErrNoActiveSubscription
	}

	// Get plan feature limit
	feature, err := s.Queries.GetPlanFeatureByKey(ctx, db.GetPlanFeatureByKeyParams{
		PlanID:     sub.PlanID,
		FeatureKey: featureKey,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}

	if feature.FeatureType != "quota" || !feature.ValueInt.Valid {
		return false, nil
	}

	limit := int(feature.ValueInt.Int32)
	if limit == -1 {
		return true, nil // Unlimited
	}

	// Get current usage
	now := time.Now()
	usage, err := s.Queries.GetUsage(ctx, db.GetUsageParams{
		WorkspaceID: wsID,
		FeatureKey:  featureKey,
		PeriodStart: now,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return true, nil // No usage yet
		}
		return false, err
	}

	return int(usage.UsageCount) < limit, nil
}

// IncrementUsage increments the usage counter for a feature
func (s *Service) IncrementUsage(ctx context.Context, workspaceID, featureKey string) error {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return fmt.Errorf("invalid workspace_id")
	}

	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	periodEnd := periodStart.AddDate(0, 1, 0)

	_, err = s.Queries.UpsertUsage(ctx, db.UpsertUsageParams{
		ID:          uuid.New(),
		WorkspaceID: wsID,
		FeatureKey:  featureKey,
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	return err
}

// GetUsageSummary returns all usage for the current period
func (s *Service) GetUsageSummary(ctx context.Context, workspaceID string) ([]UsageResponse, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("invalid workspace_id")
	}

	// Get active subscription for plan features
	sub, err := s.Queries.GetActiveSubscription(ctx, wsID)
	if err != nil {
		return nil, ErrNoActiveSubscription
	}

	// Get plan features (quotas only)
	features, err := s.Queries.GetPlanFeatures(ctx, sub.PlanID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	var resp []UsageResponse

	for _, f := range features {
		if f.FeatureType != "quota" || !f.ValueInt.Valid {
			continue
		}

		limit := int(f.ValueInt.Int32)
		usageCount := 0

		usage, err := s.Queries.GetUsage(ctx, db.GetUsageParams{
			WorkspaceID: wsID,
			FeatureKey:  f.FeatureKey,
			PeriodStart: now,
		})
		if err == nil {
			usageCount = int(usage.UsageCount)
		}

		periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		periodEnd := periodStart.AddDate(0, 1, 0)

		resp = append(resp, UsageResponse{
			FeatureKey:  f.FeatureKey,
			UsageCount:  usageCount,
			Limit:       limit,
			PeriodStart: periodStart.Format(time.RFC3339),
			PeriodEnd:   periodEnd.Format(time.RFC3339),
		})
	}

	if resp == nil {
		resp = []UsageResponse{}
	}
	return resp, nil
}

// ListAllSubscriptions returns all subscriptions (admin)
func (s *Service) ListAllSubscriptions(ctx context.Context, limit, offset int) ([]SubscriptionResponse, error) {
	if limit <= 0 {
		limit = 50
	}

	subs, err := s.Queries.ListAllSubscriptions(ctx, db.ListAllSubscriptionsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, err
	}

	var resp []SubscriptionResponse
	for i := range subs {
		r := SubscriptionResponse{
			ID:          subs[i].ID.String(),
			WorkspaceID: subs[i].WorkspaceID.String(),
			PlanID:      subs[i].PlanID.String(),
			PlanName:    &subs[i].PlanName,
			PlanSlug:    &subs[i].PlanSlug,
			Status:      subs[i].Status,
			StartedAt:   subs[i].StartedAt.Format(time.RFC3339),
			CreatedAt:   subs[i].CreatedAt.Format(time.RFC3339),
		}
		if subs[i].BillingCycle.Valid {
			r.BillingCycle = &subs[i].BillingCycle.String
		}
		if subs[i].ExpiresAt != nil {
			t := subs[i].ExpiresAt.Format(time.RFC3339)
			r.ExpiresAt = &t
		}
		if subs[i].TrialEndsAt != nil {
			t := subs[i].TrialEndsAt.Format(time.RFC3339)
			r.TrialEndsAt = &t
		}
		resp = append(resp, r)
	}
	if resp == nil {
		resp = []SubscriptionResponse{}
	}
	return resp, nil
}

// --- Helpers ---

func numericToFloat(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	f, _ := n.Float64Value()
	return f.Float64
}
