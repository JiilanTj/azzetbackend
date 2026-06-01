package claim

const (
	StatusDraft        = "DRAFT"
	StatusSubmitted    = "SUBMITTED"
	StatusUnderReview  = "UNDER_REVIEW"
	StatusApproved     = "APPROVED"
	StatusRejected     = "REJECTED"
	StatusDisputed     = "DISPUTED"

	ActorUser   = "USER"
	ActorAdmin  = "ADMIN"
	ActorSystem = "SYSTEM"

	ActionCreated        = "CREATED"
	ActionSubmitted      = "SUBMITTED"
	ActionDocumentUploaded = "DOCUMENT_UPLOADED"
	ActionAssigned       = "ASSIGNED"
	ActionApproved       = "APPROVED"
	ActionRejected       = "REJECTED"
	ActionDisputed       = "DISPUTED"
	ActionResubmitted    = "RESUBMITTED"
	ActionNoteAdded      = "NOTE_ADDED"

	DocStatusPending   = "PENDING"
	DocStatusUploaded  = "UPLOADED"
	DocStatusVerified  = "VERIFIED"
	DocStatusRejected  = "REJECTED"
)

// --- Request DTOs ---

type CreateClaimRequest struct {
	EntityID string  `json:"entity_id"`
	Notes    *string `json:"notes,omitempty"`
}

type DocumentUploadRequest struct {
	DocumentType string `json:"document_type"`
	FileName     string `json:"file_name"`
	MimeType     string `json:"mime_type"`
	FileSize     int64  `json:"file_size"`
}

type ReviewRequest struct {
	Notes           *string `json:"notes,omitempty"`
	RejectionReason *string `json:"rejection_reason,omitempty"`
}

type DisputeRequest struct {
	Reason string `json:"reason"`
}

// --- Response DTOs ---

type ClaimResponse struct {
	ID           string `json:"id"`
	EntityID     string `json:"entity_id"`
	EntityName   string `json:"entity_name"`
	EntityType   string `json:"entity_type"`
	Status       string `json:"status"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type ClaimDetailResponse struct {
	ClaimResponse
	ClaimantUserID    string             `json:"claimant_user_id"`
	ClaimantEntityID  string             `json:"claimant_entity_id"`
	ReviewerID        *string            `json:"reviewer_id,omitempty"`
	ReviewedAt        *string            `json:"reviewed_at,omitempty"`
	RejectionReason   *string            `json:"rejection_reason,omitempty"`
	DisputeReason     *string            `json:"dispute_reason,omitempty"`
	Notes             *string            `json:"notes,omitempty"`
	Documents         []DocumentResponse `json:"documents"`
	AuditLog          []AuditLogEntry    `json:"audit_log"`
}

type ClaimListResponse struct {
	ClaimResponse
	ClaimantName  string `json:"claimant_name"`
	DocumentCount int    `json:"document_count"`
}

type PresignedUploadResponse struct {
	DocumentID string `json:"document_id"`
	UploadURL  string `json:"upload_url"`
	FileKey    string `json:"file_key"`
	ExpiresIn  int    `json:"expires_in"`
}

type DocumentResponse struct {
	ID           string  `json:"id"`
	ClaimID      string  `json:"claim_id"`
	DocumentType string  `json:"document_type"`
	FileName     string  `json:"file_name"`
	FileSize     int64   `json:"file_size"`
	MimeType     string  `json:"mime_type"`
	UploadStatus string  `json:"upload_status"`
	ViewURL      *string `json:"view_url,omitempty"`
	CreatedAt    string  `json:"created_at"`
}

type AuditLogEntry struct {
	ID         string                 `json:"id"`
	ClaimID    string                 `json:"claim_id"`
	ActorID    string                 `json:"actor_id"`
	ActorType  string                 `json:"actor_type"`
	Action     string                 `json:"action"`
	OldStatus  *string                `json:"old_status,omitempty"`
	NewStatus  *string                `json:"new_status,omitempty"`
	Details    map[string]interface{} `json:"details,omitempty"`
	CreatedAt  string                 `json:"created_at"`
}
