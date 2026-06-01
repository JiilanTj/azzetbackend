package identity

const (
	StatusUnverified = "UNVERIFIED"
	StatusPending    = "PENDING"
	StatusVerified   = "VERIFIED"
	StatusRejected   = "REJECTED"
)

// --- Request DTOs ---

type AddLegalIDRequest struct {
	IDType  string `json:"id_type"`
	IDValue string `json:"id_value"`
}

type AddAliasRequest struct {
	Alias  string `json:"alias"`
	Source string `json:"source,omitempty"`
}

// --- Response DTOs ---

type VerificationResponse struct {
	EntityID        string  `json:"entity_id"`
	Status          string  `json:"status"`
	VerifiedBy      *string `json:"verified_by,omitempty"`
	VerifiedAt      *string `json:"verified_at,omitempty"`
	RejectionReason *string `json:"rejection_reason,omitempty"`
	Notes           *string `json:"notes,omitempty"`
}

type LegalIDResponse struct {
	ID         string  `json:"id"`
	EntityID   string  `json:"entity_id"`
	IDType     string  `json:"id_type"`
	IDValue    string  `json:"id_value"`
	IsVerified bool    `json:"is_verified"`
	VerifiedAt *string `json:"verified_at,omitempty"`
	CreatedAt  string  `json:"created_at"`
}

type AliasResponse struct {
	ID        string `json:"id"`
	EntityID  string `json:"entity_id"`
	Alias     string `json:"alias"`
	Source    string `json:"source"`
	CreatedAt string `json:"created_at"`
}

type FuzzyMatchResponse struct {
	ID         string  `json:"id"`
	NamaUtama  string  `json:"nama_utama"`
	EntityType string  `json:"entity_type"`
	IsShadow   bool    `json:"is_shadow"`
	MatchScore float64 `json:"match_score"`
}
