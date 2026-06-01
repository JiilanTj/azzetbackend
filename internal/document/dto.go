package document

const (
	DocTypeReceipt = "RECEIPT"
	DocTypeInvoice = "INVOICE"
	DocTypeFaktur  = "FAKTUR"
	DocTypeOther   = "OTHER"

	UploadStatusPending  = "PENDING"
	UploadStatusUploaded = "UPLOADED"
	UploadStatusFailed   = "FAILED"

	ExtractionPending    = "PENDING"
	ExtractionProcessing = "PROCESSING"
	ExtractionCompleted  = "COMPLETED"
	ExtractionFailed     = "FAILED"
	ExtractionSkipped    = "SKIPPED"

	VerificationUnverified = "UNVERIFIED"
	VerificationVerified   = "VERIFIED"
	VerificationRejected   = "REJECTED"
)

var validDocTypes = map[string]bool{
	DocTypeReceipt: true,
	DocTypeInvoice: true,
	DocTypeFaktur:  true,
	DocTypeOther:   true,
}

// --- Request DTOs ---

type UploadRequest struct {
	DocumentType string `json:"document_type"`
	FileName     string `json:"file_name"`
	MimeType     string `json:"mime_type"`
	FileSize     int64  `json:"file_size"`
}

// --- Response DTOs ---

type DocumentResponse struct {
	ID                   string                 `json:"id"`
	WorkspaceID          string                 `json:"workspace_id"`
	DocumentType         string                 `json:"document_type"`
	FileName             string                 `json:"file_name"`
	FileSize             int64                  `json:"file_size"`
	MimeType             string                 `json:"mime_type"`
	UploadStatus         string                 `json:"upload_status"`
	ExtractionStatus     string                 `json:"extraction_status"`
	VerificationStatus   string                 `json:"verification_status"`
	ExtractedData        map[string]interface{} `json:"extracted_data,omitempty"`
	ExtractionConfidence *float64               `json:"extraction_confidence,omitempty"`
	ExtractionError      *string                `json:"extraction_error,omitempty"`
	TransactionID        *string                `json:"transaction_id,omitempty"`
	ViewURL              *string                `json:"view_url,omitempty"`
	UploadedAt           *string                `json:"uploaded_at,omitempty"`
	ProcessedAt          *string                `json:"processed_at,omitempty"`
	CreatedAt            string                 `json:"created_at"`
	UpdatedAt            string                 `json:"updated_at"`
}

type PresignedUploadResponse struct {
	DocumentID string `json:"document_id"`
	UploadURL  string `json:"upload_url"`
	FileKey    string `json:"file_key"`
	ExpiresIn  int    `json:"expires_in"`
}

type DocumentListResponse struct {
	Documents []DocumentResponse `json:"documents"`
	Total     int64              `json:"total"`
}

// ExtractionResult is the structured output from OCR/AI extraction.
type ExtractionResult struct {
	VendorName      string  `json:"vendor_name"`
	VendorNPWP      string  `json:"vendor_npwp,omitempty"`
	Amount          float64 `json:"amount"`
	Date            string  `json:"date"`
	TransactionType string  `json:"transaction_type"`
	Category        string  `json:"category"`
	PaymentMethod   string  `json:"payment_method,omitempty"`
	Description     string  `json:"description,omitempty"`
	Confidence      float64 `json:"confidence"`
}
