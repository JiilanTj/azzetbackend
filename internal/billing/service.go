package billing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"codeberg.org/azzet/azzetbe/internal/db"
)

var ErrInvoiceNotFound = errors.New("invoice not found")
var ErrPaymentNotFound = errors.New("payment not found")
var ErrAlreadyPaid = errors.New("invoice already paid")

type Service struct {
	Queries *db.Queries
	Xendit  *XenditClient
}

func NewService(queries *db.Queries, xendit *XenditClient) *Service {
	return &Service{
		Queries: queries,
		Xendit:  xendit,
	}
}

// CreateInvoice creates an invoice for a subscription
func (s *Service) CreateInvoice(ctx context.Context, workspaceID, subscriptionID string, amount float64, description string) (*InvoiceResponse, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("invalid workspace_id")
	}
	subID, err := uuid.Parse(subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("invalid subscription_id")
	}

	now := time.Now()
	invoiceNumber := GenerateInvoiceNumber(now)

	var desc pgtype.Text
	if description != "" {
		desc = pgtype.Text{String: description, Valid: true}
	}

	invoice, err := s.Queries.CreateInvoice(ctx, db.CreateInvoiceParams{
		ID:             uuid.New(),
		WorkspaceID:    wsID,
		SubscriptionID: subID,
		InvoiceNumber:  invoiceNumber,
		Amount:         numericFromFloat(amount),
		Currency:       "IDR",
		Status:         InvoiceStatusPending,
		Description:    desc,
		DueDate:        now.Add(InvoiceDuration),
		CreatedAt:      now,
		UpdatedAt:      now,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create invoice: %w", err)
	}

	return InvoiceToResponse(&invoice), nil
}

// PayInvoice initiates payment for an invoice via Xendit
func (s *Service) PayInvoice(ctx context.Context, workspaceID, invoiceID string) (*PaymentResponse, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("invalid workspace_id")
	}
	invID, err := uuid.Parse(invoiceID)
	if err != nil {
		return nil, ErrInvoiceNotFound
	}

	invoice, err := s.Queries.GetInvoiceByID(ctx, invID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvoiceNotFound
		}
		return nil, err
	}

	if invoice.WorkspaceID != wsID {
		return nil, ErrInvoiceNotFound
	}
	if invoice.Status == InvoiceStatusPaid {
		return nil, ErrAlreadyPaid
	}

	now := time.Now()

	// Reuse existing pending payment when still valid
	if existing, err := s.Queries.GetPendingPaymentByInvoice(ctx, invID); err == nil {
		if existing.ExpiredAt == nil || existing.ExpiredAt.After(now) {
			return PaymentToResponse(&existing), nil
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	// Get amount as float
	amount := numericToFloat(invoice.Amount)

	// Create Xendit invoice
	expiresAt := now.Add(InvoiceDuration)

	xenditResp, err := s.Xendit.CreateInvoice(ctx, &XenditCreateInvoiceRequest{
		ExternalID:      invoice.ID.String(),
		Amount:          amount,
		Description:     invoice.InvoiceNumber,
		InvoiceDuration: int(InvoiceDuration.Seconds()),
		Currency:        "IDR",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create payment: %w", err)
	}

	// Store payment record
	var xenditID pgtype.Text
	var paymentURL pgtype.Text
	if xenditResp != nil {
		xenditID = pgtype.Text{String: xenditResp.ID, Valid: true}
		paymentURL = pgtype.Text{String: xenditResp.InvoiceURL, Valid: true}
	}

	payment, err := s.Queries.CreatePayment(ctx, db.CreatePaymentParams{
		ID:               uuid.New(),
		InvoiceID:        invID,
		WorkspaceID:      wsID,
		XenditInvoiceID:  xenditID,
		XenditPaymentUrl: paymentURL,
		Amount:           invoice.Amount,
		Currency:         "IDR",
		Status:           PaymentStatusPending,
		ExpiredAt:        &expiresAt,
		CreatedAt:        now,
		UpdatedAt:        now,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to store payment: %w", err)
	}

	return PaymentToResponse(&payment), nil
}

// HandleWebhook processes Xendit payment callback
func (s *Service) HandleWebhook(ctx context.Context, payload *XenditWebhookPayload) error {
	if payload.ID == "" {
		return fmt.Errorf("invalid webhook payload")
	}

	// Find payment by Xendit invoice ID
	payment, err := s.Queries.GetPaymentByXenditID(ctx, pgtype.Text{String: payload.ID, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("payment not found for xendit_id: %s", payload.ID)
		}
		return err
	}

	now := time.Now()

	invoice, err := s.Queries.GetInvoiceByID(ctx, payment.InvoiceID)
	if err != nil {
		return fmt.Errorf("invoice not found")
	}
	if payload.ExternalID != "" && payload.ExternalID != invoice.ID.String() {
		return fmt.Errorf("webhook external_id mismatch")
	}
	invoiceAmount := numericToFloat(invoice.Amount)
	paidAmount := payload.PaidAmount
	if paidAmount == 0 {
		paidAmount = payload.Amount
	}
	if paidAmount > 0 && paidAmount != invoiceAmount {
		return fmt.Errorf("webhook amount mismatch")
	}

	switch payload.Status {
	case "PAID", "SETTLED":
		// Mark payment as paid
		callbackData, _ := json.Marshal(payload)
		method := payload.PaymentMethod
		if payload.PaymentChannel != "" {
			method = payload.PaymentChannel
		}

		err = s.Queries.UpdatePaymentStatus(ctx, db.UpdatePaymentStatusParams{
			ID:                 payment.ID,
			Status:             PaymentStatusPaid,
			PaymentMethod:      pgtype.Text{String: method, Valid: method != ""},
			PaidAt:             &now,
			FailureReason:      pgtype.Text{Valid: false},
			XenditCallbackData: callbackData,
		})
		if err != nil {
			return err
		}

		// Mark invoice as paid
		if err := s.Queries.MarkInvoicePaid(ctx, payment.InvoiceID); err != nil {
			return fmt.Errorf("failed to mark invoice paid: %w", err)
		}

		// Activate subscription
		if err := s.Queries.UpdateSubscriptionStatus(ctx, db.UpdateSubscriptionStatusParams{
			ID:     invoice.SubscriptionID,
			Status: "active",
		}); err != nil {
			return fmt.Errorf("failed to activate subscription: %w", err)
		}

	case "EXPIRED":
		callbackData, _ := json.Marshal(payload)
		err = s.Queries.UpdatePaymentStatus(ctx, db.UpdatePaymentStatusParams{
			ID:                 payment.ID,
			Status:             PaymentStatusExpired,
			PaymentMethod:      pgtype.Text{Valid: false},
			PaidAt:             nil,
			FailureReason:      pgtype.Text{String: "payment expired", Valid: true},
			XenditCallbackData: callbackData,
		})
		if err != nil {
			return err
		}

		// Mark invoice as expired
		_ = s.Queries.UpdateInvoiceStatus(ctx, db.UpdateInvoiceStatusParams{
			ID:     payment.InvoiceID,
			Status: InvoiceStatusExpired,
		})

	case "FAILED":
		callbackData, _ := json.Marshal(payload)
		err = s.Queries.UpdatePaymentStatus(ctx, db.UpdatePaymentStatusParams{
			ID:                 payment.ID,
			Status:             PaymentStatusFailed,
			PaymentMethod:      pgtype.Text{Valid: false},
			PaidAt:             nil,
			FailureReason:      pgtype.Text{String: "payment failed", Valid: true},
			XenditCallbackData: callbackData,
		})
		if err != nil {
			return err
		}

		_ = s.Queries.UpdateInvoiceStatus(ctx, db.UpdateInvoiceStatusParams{
			ID:     payment.InvoiceID,
			Status: InvoiceStatusFailed,
		})
	}

	return nil
}

// GetInvoice returns an invoice by ID
func (s *Service) GetInvoice(ctx context.Context, workspaceID, invoiceID string) (*InvoiceResponse, error) {
	invID, err := uuid.Parse(invoiceID)
	if err != nil {
		return nil, ErrInvoiceNotFound
	}

	invoice, err := s.Queries.GetInvoiceByID(ctx, invID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvoiceNotFound
		}
		return nil, err
	}

	wsID, _ := uuid.Parse(workspaceID)
	if invoice.WorkspaceID != wsID {
		return nil, ErrInvoiceNotFound
	}

	return InvoiceToResponse(&invoice), nil
}

// ListInvoices returns invoices for a workspace
func (s *Service) ListInvoices(ctx context.Context, workspaceID string, limit, offset int) ([]InvoiceResponse, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("invalid workspace_id")
	}

	if limit <= 0 {
		limit = 20
	}

	invoices, err := s.Queries.ListInvoicesByWorkspace(ctx, db.ListInvoicesByWorkspaceParams{
		WorkspaceID: wsID,
		Limit:       int32(limit),
		Offset:      int32(offset),
	})
	if err != nil {
		return nil, err
	}

	var resp []InvoiceResponse
	for i := range invoices {
		resp = append(resp, *InvoiceToResponse(&invoices[i]))
	}
	if resp == nil {
		resp = []InvoiceResponse{}
	}
	return resp, nil
}

// ListPayments returns payments for a workspace
func (s *Service) ListPayments(ctx context.Context, workspaceID string, limit, offset int) ([]PaymentResponse, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("invalid workspace_id")
	}

	if limit <= 0 {
		limit = 20
	}

	payments, err := s.Queries.ListPaymentsByWorkspace(ctx, db.ListPaymentsByWorkspaceParams{
		WorkspaceID: wsID,
		Limit:       int32(limit),
		Offset:      int32(offset),
	})
	if err != nil {
		return nil, err
	}

	var resp []PaymentResponse
	for i := range payments {
		resp = append(resp, *PaymentToResponse(&payments[i]))
	}
	if resp == nil {
		resp = []PaymentResponse{}
	}
	return resp, nil
}

// ListAllInvoices returns all invoices (admin)
func (s *Service) ListAllInvoices(ctx context.Context, limit, offset int) ([]InvoiceResponse, error) {
	if limit <= 0 {
		limit = 50
	}

	invoices, err := s.Queries.ListAllInvoices(ctx, db.ListAllInvoicesParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, err
	}

	var resp []InvoiceResponse
	for i := range invoices {
		r := InvoiceResponse{
			ID:             invoices[i].ID.String(),
			WorkspaceID:    invoices[i].WorkspaceID.String(),
			SubscriptionID: invoices[i].SubscriptionID.String(),
			InvoiceNumber:  invoices[i].InvoiceNumber,
			Amount:         numericToFloat(invoices[i].Amount),
			Currency:       invoices[i].Currency,
			Status:         invoices[i].Status,
			DueDate:        invoices[i].DueDate.Format(time.RFC3339),
			CreatedAt:      invoices[i].CreatedAt.Format(time.RFC3339),
		}
		if invoices[i].Description.Valid {
			r.Description = &invoices[i].Description.String
		}
		if invoices[i].PaidAt != nil {
			t := invoices[i].PaidAt.Format(time.RFC3339)
			r.PaidAt = &t
		}
		resp = append(resp, r)
	}
	if resp == nil {
		resp = []InvoiceResponse{}
	}
	return resp, nil
}

// --- Helpers ---

func InvoiceToResponse(i *db.Invoice) *InvoiceResponse {
	resp := &InvoiceResponse{
		ID:             i.ID.String(),
		WorkspaceID:    i.WorkspaceID.String(),
		SubscriptionID: i.SubscriptionID.String(),
		InvoiceNumber:  i.InvoiceNumber,
		Amount:         numericToFloat(i.Amount),
		Currency:       i.Currency,
		Status:         i.Status,
		DueDate:        i.DueDate.Format(time.RFC3339),
		CreatedAt:      i.CreatedAt.Format(time.RFC3339),
	}
	if i.Description.Valid {
		resp.Description = &i.Description.String
	}
	if i.PaidAt != nil {
		t := i.PaidAt.Format(time.RFC3339)
		resp.PaidAt = &t
	}
	return resp
}

func PaymentToResponse(p *db.Payment) *PaymentResponse {
	resp := &PaymentResponse{
		ID:        p.ID.String(),
		InvoiceID: p.InvoiceID.String(),
		Amount:    numericToFloat(p.Amount),
		Currency:  p.Currency,
		Status:    p.Status,
		CreatedAt: p.CreatedAt.Format(time.RFC3339),
	}
	if p.PaymentMethod.Valid {
		resp.PaymentMethod = &p.PaymentMethod.String
	}
	if p.XenditPaymentUrl.Valid {
		resp.PaymentURL = &p.XenditPaymentUrl.String
	}
	if p.PaidAt != nil {
		t := p.PaidAt.Format(time.RFC3339)
		resp.PaidAt = &t
	}
	if p.ExpiredAt != nil {
		t := p.ExpiredAt.Format(time.RFC3339)
		resp.ExpiresAt = &t
	}
	return resp
}

func numericFromFloat(f float64) pgtype.Numeric {
	var n pgtype.Numeric
	n.Scan(fmt.Sprintf("%.2f", f))
	return n
}

func numericToFloat(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	f, _ := n.Float64Value()
	return f.Float64
}
