package plan

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"codeberg.org/azzet/azzetbe/internal/db"
)

var ErrPlanNotFound = errors.New("plan not found")

type Service struct {
	Queries *db.Queries
}

func NewService(queries *db.Queries) *Service {
	return &Service{Queries: queries}
}

// CreatePlan creates a new plan
func (s *Service) CreatePlan(ctx context.Context, req *CreatePlanRequest) (*PlanResponse, error) {
	if !IsValidPlanType(req.Type) {
		return nil, fmt.Errorf("invalid plan type")
	}
	if req.Name == "" || req.Slug == "" {
		return nil, fmt.Errorf("name and slug are required")
	}
	if req.IsTrial && req.TrialDays <= 0 {
		return nil, fmt.Errorf("trial_days must be > 0 when is_trial is true")
	}

	exists, err := s.Queries.ExistsPlanBySlug(ctx, req.Slug)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("plan with slug '%s' already exists", req.Slug)
	}

	now := time.Now()
	var desc pgtype.Text
	if req.Description != nil {
		desc = pgtype.Text{String: *req.Description, Valid: true}
	}

	p, err := s.Queries.CreatePlan(ctx, db.CreatePlanParams{
		ID:           uuid.New(),
		Name:         req.Name,
		Slug:         req.Slug,
		Description:  desc,
		Type:         req.Type,
		PriceMonthly: numericFromFloat(req.PriceMonthly),
		PriceYearly:  numericFromFloat(req.PriceYearly),
		IsTrial:      req.IsTrial,
		TrialDays:    int32(req.TrialDays),
		Tier:         int32(req.Tier),
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create plan: %w", err)
	}

	return s.planToResponse(&p, nil), nil
}

// GetPlanByID returns a plan with its features
func (s *Service) GetPlanByID(ctx context.Context, planID string) (*PlanResponse, error) {
	id, err := uuid.Parse(planID)
	if err != nil {
		return nil, ErrPlanNotFound
	}

	p, err := s.Queries.GetPlanByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPlanNotFound
		}
		return nil, err
	}

	features, err := s.Queries.GetPlanFeatures(ctx, id)
	if err != nil {
		return nil, err
	}

	return s.planToResponse(&p, features), nil
}

// GetPlanBySlug returns a plan with its features (public)
func (s *Service) GetPlanBySlug(ctx context.Context, slug string) (*PlanResponse, error) {
	p, err := s.Queries.GetPlanBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPlanNotFound
		}
		return nil, err
	}

	features, err := s.Queries.GetPlanFeatures(ctx, p.ID)
	if err != nil {
		return nil, err
	}

	return s.planToResponse(&p, features), nil
}

// ListPlans returns all active plans (public)
func (s *Service) ListPlans(ctx context.Context) ([]PlanListResponse, error) {
	plans, err := s.Queries.ListPlans(ctx)
	if err != nil {
		return nil, err
	}

	var resp []PlanListResponse
	for i := range plans {
		resp = append(resp, planToListResponse(&plans[i]))
	}
	if resp == nil {
		resp = []PlanListResponse{}
	}
	return resp, nil
}

// ListAllPlans returns all plans including inactive with features (admin)
func (s *Service) ListAllPlans(ctx context.Context) ([]PlanResponse, error) {
	plans, err := s.Queries.ListAllPlans(ctx)
	if err != nil {
		return nil, err
	}

	var resp []PlanResponse
	for i := range plans {
		features, err := s.Queries.GetPlanFeatures(ctx, plans[i].ID)
		if err != nil {
			return nil, err
		}
		resp = append(resp, *s.planToResponse(&plans[i], features))
	}
	if resp == nil {
		resp = []PlanResponse{}
	}
	return resp, nil
}

// UpdatePlan updates a plan
func (s *Service) UpdatePlan(ctx context.Context, planID string, req *UpdatePlanRequest) error {
	id, err := uuid.Parse(planID)
	if err != nil {
		return ErrPlanNotFound
	}

	p, err := s.Queries.GetPlanByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrPlanNotFound
		}
		return err
	}

	// Apply updates
	name := p.Name
	slug := p.Slug
	desc := p.Description
	planType := p.Type
	priceMonthly := p.PriceMonthly
	priceYearly := p.PriceYearly
	isTrial := p.IsTrial
	trialDays := p.TrialDays
	tier := p.Tier
	isActive := p.IsActive

	if req.Name != nil {
		name = *req.Name
	}
	if req.Slug != nil {
		slug = *req.Slug
	}
	if req.Description != nil {
		desc = pgtype.Text{String: *req.Description, Valid: true}
	}
	if req.Type != nil {
		if !IsValidPlanType(*req.Type) {
			return fmt.Errorf("invalid plan type")
		}
		planType = *req.Type
	}
	if req.PriceMonthly != nil {
		priceMonthly = numericFromFloat(*req.PriceMonthly)
	}
	if req.PriceYearly != nil {
		priceYearly = numericFromFloat(*req.PriceYearly)
	}
	if req.IsTrial != nil {
		isTrial = *req.IsTrial
	}
	if req.TrialDays != nil {
		trialDays = int32(*req.TrialDays)
	}
	if req.Tier != nil {
		tier = int32(*req.Tier)
	}
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	if isTrial && trialDays <= 0 {
		return fmt.Errorf("trial_days must be > 0 when is_trial is true")
	}

	return s.Queries.UpdatePlan(ctx, db.UpdatePlanParams{
		ID:           id,
		Name:         name,
		Slug:         slug,
		Description:  desc,
		Type:         planType,
		PriceMonthly: priceMonthly,
		PriceYearly:  priceYearly,
		IsTrial:      isTrial,
		TrialDays:    trialDays,
		Tier:         tier,
		IsActive:     isActive,
	})
}

// DeletePlan soft-deletes a plan (sets is_active = false)
func (s *Service) DeletePlan(ctx context.Context, planID string) error {
	id, err := uuid.Parse(planID)
	if err != nil {
		return ErrPlanNotFound
	}
	return s.Queries.DeletePlan(ctx, id)
}

// SetFeature adds or updates a feature on a plan
func (s *Service) SetFeature(ctx context.Context, planID string, req *SetFeatureRequest) (*FeatureResponse, error) {
	id, err := uuid.Parse(planID)
	if err != nil {
		return nil, ErrPlanNotFound
	}

	// Verify plan exists
	_, err = s.Queries.GetPlanByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPlanNotFound
		}
		return nil, err
	}

	if req.FeatureKey == "" {
		return nil, fmt.Errorf("feature_key is required")
	}
	if !IsValidFeatureType(req.FeatureType) {
		return nil, fmt.Errorf("invalid feature_type")
	}

	var valueBool pgtype.Bool
	var valueInt pgtype.Int4
	var valueText pgtype.Text

	switch req.FeatureType {
	case FeatureTypeBoolean:
		if req.ValueBool == nil {
			return nil, fmt.Errorf("value_bool is required for boolean features")
		}
		valueBool = pgtype.Bool{Bool: *req.ValueBool, Valid: true}
	case FeatureTypeQuota:
		if req.ValueInt == nil {
			return nil, fmt.Errorf("value_int is required for quota features")
		}
		valueInt = pgtype.Int4{Int32: int32(*req.ValueInt), Valid: true}
	case FeatureTypeTier:
		if req.ValueText == nil {
			return nil, fmt.Errorf("value_text is required for tier features")
		}
		valueText = pgtype.Text{String: *req.ValueText, Valid: true}
	}

	f, err := s.Queries.UpsertPlanFeature(ctx, db.UpsertPlanFeatureParams{
		ID:          uuid.New(),
		PlanID:      id,
		FeatureKey:  req.FeatureKey,
		FeatureType: req.FeatureType,
		ValueBool:   valueBool,
		ValueInt:    valueInt,
		ValueText:   valueText,
		CreatedAt:   time.Now(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to set feature: %w", err)
	}

	return featureToResponse(&f), nil
}

// RemoveFeature removes a feature from a plan
func (s *Service) RemoveFeature(ctx context.Context, planID, featureKey string) error {
	id, err := uuid.Parse(planID)
	if err != nil {
		return ErrPlanNotFound
	}
	return s.Queries.DeletePlanFeature(ctx, db.DeletePlanFeatureParams{
		PlanID:     id,
		FeatureKey: featureKey,
	})
}

// --- Helpers ---

func (s *Service) planToResponse(p *db.Plan, features []db.PlanFeature) *PlanResponse {
	var desc *string
	if p.Description.Valid {
		desc = &p.Description.String
	}

	var featureResp []FeatureResponse
	for i := range features {
		featureResp = append(featureResp, *featureToResponse(&features[i]))
	}
	if featureResp == nil {
		featureResp = []FeatureResponse{}
	}

	return &PlanResponse{
		ID:           p.ID.String(),
		Name:         p.Name,
		Slug:         p.Slug,
		Description:  desc,
		Type:         p.Type,
		PriceMonthly: numericToFloat(p.PriceMonthly),
		PriceYearly:  numericToFloat(p.PriceYearly),
		IsTrial:      p.IsTrial,
		TrialDays:    int(p.TrialDays),
		Tier:         int(p.Tier),
		IsActive:     p.IsActive,
		Features:     featureResp,
		CreatedAt:    p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    p.UpdatedAt.Format(time.RFC3339),
	}
}

func planToListResponse(p *db.Plan) PlanListResponse {
	var desc *string
	if p.Description.Valid {
		desc = &p.Description.String
	}

	return PlanListResponse{
		ID:           p.ID.String(),
		Name:         p.Name,
		Slug:         p.Slug,
		Description:  desc,
		Type:         p.Type,
		PriceMonthly: numericToFloat(p.PriceMonthly),
		PriceYearly:  numericToFloat(p.PriceYearly),
		IsTrial:      p.IsTrial,
		TrialDays:    int(p.TrialDays),
		Tier:         int(p.Tier),
		IsActive:     p.IsActive,
		CreatedAt:    p.CreatedAt.Format(time.RFC3339),
	}
}

func featureToResponse(f *db.PlanFeature) *FeatureResponse {
	resp := &FeatureResponse{
		FeatureKey:  f.FeatureKey,
		FeatureType: f.FeatureType,
	}

	if f.ValueBool.Valid {
		resp.ValueBool = &f.ValueBool.Bool
	}
	if f.ValueInt.Valid {
		v := int(f.ValueInt.Int32)
		resp.ValueInt = &v
	}
	if f.ValueText.Valid {
		resp.ValueText = &f.ValueText.String
	}

	return resp
}

// numericFromFloat converts float64 to pgtype.Numeric
func numericFromFloat(f float64) pgtype.Numeric {
	var n pgtype.Numeric
	n.Scan(fmt.Sprintf("%.2f", f))
	return n
}

// numericToFloat converts pgtype.Numeric to float64
func numericToFloat(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	f, _ := n.Float64Value()
	return f.Float64
}
