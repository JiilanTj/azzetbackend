package plan

// --- Request DTOs ---

// CreatePlanRequest represents plan creation payload
// @Description Create a new plan (SUPER_ADMIN/ENGINEER only)
type CreatePlanRequest struct {
	Name         string  `json:"name" example:"Professional"`
	Slug         string  `json:"slug" example:"professional"`
	Description  *string `json:"description,omitempty" example:"Best for growing businesses"`
	Type         string  `json:"type" example:"paid" enums:"free,paid"`
	PriceMonthly float64 `json:"price_monthly" example:"299000"`
	PriceYearly  float64 `json:"price_yearly" example:"2990000"`
	IsTrial      bool    `json:"is_trial" example:"true"`
	TrialDays    int     `json:"trial_days" example:"14"`
	Tier         int     `json:"tier" example:"2"`
}

// UpdatePlanRequest represents plan update payload
// @Description Update an existing plan
type UpdatePlanRequest struct {
	Name         *string  `json:"name,omitempty" example:"Professional Plus"`
	Slug         *string  `json:"slug,omitempty" example:"professional-plus"`
	Description  *string  `json:"description,omitempty" example:"Updated description"`
	Type         *string  `json:"type,omitempty" example:"paid" enums:"free,paid"`
	PriceMonthly *float64 `json:"price_monthly,omitempty" example:"399000"`
	PriceYearly  *float64 `json:"price_yearly,omitempty" example:"3990000"`
	IsTrial      *bool    `json:"is_trial,omitempty" example:"true"`
	TrialDays    *int     `json:"trial_days,omitempty" example:"14"`
	Tier         *int     `json:"tier,omitempty" example:"3"`
	IsActive     *bool    `json:"is_active,omitempty" example:"true"`
}

// SetFeatureRequest represents adding/updating a feature on a plan
// @Description Set a feature permission on a plan. Use value_bool for boolean features, value_int for quotas (-1 = unlimited).
type SetFeatureRequest struct {
	FeatureKey  string  `json:"feature_key" example:"max_transactions_monthly"`
	FeatureType string  `json:"feature_type" example:"quota" enums:"boolean,quota,tier"`
	ValueBool   *bool   `json:"value_bool,omitempty" example:"true"`
	ValueInt    *int    `json:"value_int,omitempty" example:"1000"`
	ValueText   *string `json:"value_text,omitempty" example:"advanced"`
}

// --- Response DTOs ---

// PlanResponse represents a plan with its features
// @Description Plan information with feature list
type PlanResponse struct {
	ID           string            `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name         string            `json:"name" example:"Professional"`
	Slug         string            `json:"slug" example:"professional"`
	Description  *string           `json:"description,omitempty" example:"Best for growing businesses"`
	Type         string            `json:"type" example:"paid"`
	PriceMonthly float64           `json:"price_monthly" example:"299000"`
	PriceYearly  float64           `json:"price_yearly" example:"2990000"`
	IsTrial      bool              `json:"is_trial" example:"true"`
	TrialDays    int               `json:"trial_days" example:"14"`
	Tier         int               `json:"tier" example:"2"`
	IsActive     bool              `json:"is_active" example:"true"`
	Features     []FeatureResponse `json:"features"`
	CreatedAt    string            `json:"created_at" example:"2026-05-19T10:00:00Z"`
	UpdatedAt    string            `json:"updated_at" example:"2026-05-19T10:00:00Z"`
}

// PlanListResponse represents a plan without features (for listing)
// @Description Plan summary for listing
type PlanListResponse struct {
	ID           string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name         string  `json:"name" example:"Professional"`
	Slug         string  `json:"slug" example:"professional"`
	Description  *string `json:"description,omitempty" example:"Best for growing businesses"`
	Type         string  `json:"type" example:"paid"`
	PriceMonthly float64 `json:"price_monthly" example:"299000"`
	PriceYearly  float64 `json:"price_yearly" example:"2990000"`
	IsTrial      bool    `json:"is_trial" example:"true"`
	TrialDays    int     `json:"trial_days" example:"14"`
	Tier         int     `json:"tier" example:"2"`
	IsActive     bool    `json:"is_active" example:"true"`
	CreatedAt    string  `json:"created_at" example:"2026-05-19T10:00:00Z"`
}

// FeatureResponse represents a single plan feature
// @Description Feature permission/quota for a plan
type FeatureResponse struct {
	FeatureKey  string  `json:"feature_key" example:"max_transactions_monthly"`
	FeatureType string  `json:"feature_type" example:"quota"`
	ValueBool   *bool   `json:"value_bool,omitempty" example:"true"`
	ValueInt    *int    `json:"value_int,omitempty" example:"1000"`
	ValueText   *string `json:"value_text,omitempty" example:"advanced"`
}

// MessageResponse represents a simple message
// @Description Simple message response
type MessageResponse struct {
	Message string `json:"message" example:"Operation successful"`
}

// --- Constants ---

const (
	PlanTypeFree = "free"
	PlanTypePaid = "paid"

	FeatureTypeBoolean = "boolean"
	FeatureTypeQuota   = "quota"
	FeatureTypeTier    = "tier"

	QuotaUnlimited = -1
)

var ValidPlanTypes = []string{PlanTypeFree, PlanTypePaid}
var ValidFeatureTypes = []string{FeatureTypeBoolean, FeatureTypeQuota, FeatureTypeTier}

func IsValidPlanType(t string) bool {
	for _, v := range ValidPlanTypes {
		if v == t {
			return true
		}
	}
	return false
}

func IsValidFeatureType(t string) bool {
	for _, v := range ValidFeatureTypes {
		if v == t {
			return true
		}
	}
	return false
}
