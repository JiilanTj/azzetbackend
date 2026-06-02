package billing

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// --- Request DTOs ---

// CreatePaymentRequest represents a payment initiation
// @Description Initiate payment for a subscription invoice
type CreatePaymentRequest struct {
	InvoiceID string `json:"invoice_id" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// --- Response DTOs ---

// InvoiceResponse represents an invoice
// @Description Invoice information
type InvoiceResponse struct {
	ID             string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	WorkspaceID    string  `json:"workspace_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	SubscriptionID string  `json:"subscription_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	InvoiceNumber  string  `json:"invoice_number" example:"INV-2026-0001"`
	Amount         float64 `json:"amount" example:"299000"`
	Currency       string  `json:"currency" example:"IDR"`
	Status         string  `json:"status" example:"pending" enums:"pending,paid,failed,expired,refunded"`
	Description    *string `json:"description,omitempty" example:"Professional Plan - Monthly"`
	DueDate        string  `json:"due_date" example:"2026-06-20T00:00:00Z"`
	PaidAt         *string `json:"paid_at,omitempty" example:"2026-05-20T10:00:00Z"`
	CreatedAt      string  `json:"created_at" example:"2026-05-20T10:00:00Z"`
}

// PaymentResponse represents a payment attempt
// @Description Payment attempt information
type PaymentResponse struct {
	ID            string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	InvoiceID     string  `json:"invoice_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Amount        float64 `json:"amount" example:"299000"`
	Currency      string  `json:"currency" example:"IDR"`
	Status        string  `json:"status" example:"pending" enums:"pending,paid,failed,expired,refunded"`
	PaymentMethod *string `json:"payment_method,omitempty" example:"BANK_TRANSFER"`
	PaymentURL    *string `json:"payment_url,omitempty" example:"https://checkout.xendit.co/..."`
	PaidAt        *string `json:"paid_at,omitempty" example:"2026-05-20T10:00:00Z"`
	ExpiresAt     *string `json:"expires_at,omitempty" example:"2026-05-21T10:00:00Z"`
	CreatedAt     string  `json:"created_at" example:"2026-05-20T10:00:00Z"`
}

// MessageResponse represents a simple message
// @Description Simple message response
type MessageResponse struct {
	Message string `json:"message" example:"Operation successful"`
}

// --- Constants ---

const (
	InvoiceStatusPending  = "pending"
	InvoiceStatusPaid     = "paid"
	InvoiceStatusFailed   = "failed"
	InvoiceStatusExpired  = "expired"
	InvoiceStatusRefunded = "refunded"

	PaymentStatusPending  = "pending"
	PaymentStatusPaid     = "paid"
	PaymentStatusFailed   = "failed"
	PaymentStatusExpired  = "expired"
	PaymentStatusRefunded = "refunded"

	InvoiceDuration = 24 * time.Hour // 24 hours to pay
)

func GenerateInvoiceNumber(now time.Time) string {
	suffix := uuid.New().String()[:8]
	return now.Format("INV-20060102") + "-" + strings.ToUpper(suffix)
}
