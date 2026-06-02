package billing_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"codeberg.org/azzet/azzetbe/internal/billing"
	"codeberg.org/azzet/azzetbe/internal/db"
)

func TestInvoiceToResponse_Pending(t *testing.T) {
	now := time.Now()
	dueDate := now.Add(24 * time.Hour)

	invoice := &db.Invoice{
		ID:             uuid.New(),
		WorkspaceID:    uuid.New(),
		SubscriptionID: uuid.New(),
		InvoiceNumber:  "INV-2026-05-0001",
		Amount:         numericFromFloat(299000),
		Currency:       "IDR",
		Status:         billing.InvoiceStatusPending,
		Description:    pgtype.Text{String: "Professional Plan - Monthly", Valid: true},
		DueDate:        dueDate,
		PaidAt:         nil,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	resp := billing.InvoiceToResponse(invoice)

	if resp.ID != invoice.ID.String() {
		t.Fatalf("expected ID '%s', got '%s'", invoice.ID.String(), resp.ID)
	}
	if resp.InvoiceNumber != "INV-2026-05-0001" {
		t.Fatalf("expected invoice_number 'INV-2026-05-0001', got '%s'", resp.InvoiceNumber)
	}
	if resp.Status != billing.InvoiceStatusPending {
		t.Fatalf("expected status 'pending', got '%s'", resp.Status)
	}
	if resp.Description == nil || *resp.Description != "Professional Plan - Monthly" {
		t.Fatalf("expected description, got %v", resp.Description)
	}
	if resp.PaidAt != nil {
		t.Fatalf("expected nil paid_at, got %v", resp.PaidAt)
	}
	if resp.Currency != "IDR" {
		t.Fatalf("expected currency 'IDR', got '%s'", resp.Currency)
	}
}

func TestInvoiceToResponse_Paid(t *testing.T) {
	now := time.Now()
	paidAt := now.Add(-1 * time.Hour)

	invoice := &db.Invoice{
		ID:             uuid.New(),
		WorkspaceID:    uuid.New(),
		SubscriptionID: uuid.New(),
		InvoiceNumber:  "INV-2026-05-0002",
		Amount:         numericFromFloat(799000),
		Currency:       "IDR",
		Status:         billing.InvoiceStatusPaid,
		Description:    pgtype.Text{Valid: false},
		DueDate:        now,
		PaidAt:         &paidAt,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	resp := billing.InvoiceToResponse(invoice)

	if resp.Status != billing.InvoiceStatusPaid {
		t.Fatalf("expected status 'paid', got '%s'", resp.Status)
	}
	if resp.PaidAt == nil {
		t.Fatal("expected non-nil paid_at")
	}
	if resp.Description != nil {
		t.Fatalf("expected nil description, got %v", resp.Description)
	}
}

func TestPaymentToResponse(t *testing.T) {
	now := time.Now()
	expiresAt := now.Add(24 * time.Hour)

	payment := &db.Payment{
		ID:               uuid.New(),
		InvoiceID:        uuid.New(),
		WorkspaceID:      uuid.New(),
		XenditInvoiceID:  pgtype.Text{String: "xendit-123", Valid: true},
		XenditPaymentUrl: pgtype.Text{String: "https://checkout.xendit.co/abc", Valid: true},
		Amount:           numericFromFloat(299000),
		Currency:         "IDR",
		Status:           billing.PaymentStatusPending,
		PaymentMethod:    pgtype.Text{Valid: false},
		PaidAt:           nil,
		ExpiredAt:        &expiresAt,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	resp := billing.PaymentToResponse(payment)

	if resp.ID != payment.ID.String() {
		t.Fatalf("expected ID '%s', got '%s'", payment.ID.String(), resp.ID)
	}
	if resp.Status != billing.PaymentStatusPending {
		t.Fatalf("expected status 'pending', got '%s'", resp.Status)
	}
	if resp.PaymentURL == nil || *resp.PaymentURL != "https://checkout.xendit.co/abc" {
		t.Fatalf("expected payment_url, got %v", resp.PaymentURL)
	}
	if resp.ExpiresAt == nil {
		t.Fatal("expected non-nil expires_at")
	}
	if resp.PaymentMethod != nil {
		t.Fatalf("expected nil payment_method, got %v", resp.PaymentMethod)
	}
	if resp.PaidAt != nil {
		t.Fatalf("expected nil paid_at, got %v", resp.PaidAt)
	}
}

func TestPaymentToResponse_Paid(t *testing.T) {
	now := time.Now()
	paidAt := now.Add(-30 * time.Minute)

	payment := &db.Payment{
		ID:               uuid.New(),
		InvoiceID:        uuid.New(),
		WorkspaceID:      uuid.New(),
		XenditInvoiceID:  pgtype.Text{String: "xendit-456", Valid: true},
		XenditPaymentUrl: pgtype.Text{String: "https://checkout.xendit.co/def", Valid: true},
		Amount:           numericFromFloat(299000),
		Currency:         "IDR",
		Status:           billing.PaymentStatusPaid,
		PaymentMethod:    pgtype.Text{String: "BANK_TRANSFER", Valid: true},
		PaidAt:           &paidAt,
		ExpiredAt:        nil,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	resp := billing.PaymentToResponse(payment)

	if resp.Status != billing.PaymentStatusPaid {
		t.Fatalf("expected status 'paid', got '%s'", resp.Status)
	}
	if resp.PaymentMethod == nil || *resp.PaymentMethod != "BANK_TRANSFER" {
		t.Fatalf("expected payment_method 'BANK_TRANSFER', got %v", resp.PaymentMethod)
	}
	if resp.PaidAt == nil {
		t.Fatal("expected non-nil paid_at")
	}
}

func TestBillingConstants(t *testing.T) {
	if billing.InvoiceStatusPending != "pending" {
		t.Fatalf("expected 'pending', got '%s'", billing.InvoiceStatusPending)
	}
	if billing.InvoiceStatusPaid != "paid" {
		t.Fatalf("expected 'paid', got '%s'", billing.InvoiceStatusPaid)
	}
	if billing.InvoiceStatusFailed != "failed" {
		t.Fatalf("expected 'failed', got '%s'", billing.InvoiceStatusFailed)
	}
	if billing.InvoiceStatusExpired != "expired" {
		t.Fatalf("expected 'expired', got '%s'", billing.InvoiceStatusExpired)
	}
	if billing.PaymentStatusPending != "pending" {
		t.Fatalf("expected 'pending', got '%s'", billing.PaymentStatusPending)
	}
	if billing.PaymentStatusPaid != "paid" {
		t.Fatalf("expected 'paid', got '%s'", billing.PaymentStatusPaid)
	}
}

func TestGenerateInvoiceNumber(t *testing.T) {
	num := billing.GenerateInvoiceNumber(time.Now())
	if num == "" {
		t.Fatal("expected non-empty invoice number")
	}
	// Should contain "INV-"
	if len(num) < 10 {
		t.Fatalf("invoice number too short: '%s'", num)
	}
}

func TestXenditClient_VerifyWebhookToken(t *testing.T) {
	client := billing.NewXenditClient("api-key", "webhook-secret", "", "", "")

	if !client.VerifyWebhookToken("webhook-secret") {
		t.Fatal("expected valid token to pass")
	}
	if client.VerifyWebhookToken("wrong-token") {
		t.Fatal("expected invalid token to fail")
	}
	if client.VerifyWebhookToken("") {
		t.Fatal("expected empty token to fail")
	}
}

func TestXenditClient_VerifyWebhookToken_NoSecret(t *testing.T) {
	client := billing.NewXenditClient("api-key", "", "", "", "")

	if client.VerifyWebhookToken("any-token") {
		t.Fatal("expected verification to fail when no secret configured")
	}
}

// Helper
func numericFromFloat(f float64) pgtype.Numeric {
	var n pgtype.Numeric
	n.Scan(fmt.Sprintf("%.2f", f))
	return n
}
