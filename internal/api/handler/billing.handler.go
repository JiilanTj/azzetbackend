package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"codeberg.org/azzet/azzetbe/internal/api/middleware"
	"codeberg.org/azzet/azzetbe/internal/billing"
	"codeberg.org/azzet/azzetbe/internal/shared"
)

type BillingHandler struct {
	Service *billing.Service
}

func NewBillingHandler(service *billing.Service) *BillingHandler {
	return &BillingHandler{Service: service}
}

// ListInvoices godoc
// @Summary      List invoices
// @Description  Returns all invoices for the current workspace.
// @Tags         Billing
// @Produce      json
// @Security     BearerAuth
// @Param        X-Workspace-ID  header    string  true   "Workspace ID"
// @Param        limit           query     int     false  "Limit (default 20)"
// @Param        offset          query     int     false  "Offset (default 0)"
// @Success      200             {object}  shared.APIResponse{data=[]billing.InvoiceResponse}
// @Failure      401             {object}  shared.ErrorResponse
// @Router       /billing/invoices [get]
func (h *BillingHandler) ListInvoices(w http.ResponseWriter, r *http.Request) {
	workspaceID := middleware.GetWorkspaceID(r.Context())

	limit, offset := 20, 0
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	invoices, err := h.Service.ListInvoices(r.Context(), workspaceID, limit, offset)
	if err != nil {
		shared.InternalError(w, r, "billing", "failed to list invoices")
		return
	}

	shared.OK(w, r, invoices)
}

// GetInvoice godoc
// @Summary      Get invoice
// @Description  Returns invoice detail by ID.
// @Tags         Billing
// @Produce      json
// @Security     BearerAuth
// @Param        X-Workspace-ID  header    string  true  "Workspace ID"
// @Param        id              path      string  true  "Invoice ID (UUID)"
// @Success      200             {object}  shared.APIResponse{data=billing.InvoiceResponse}
// @Failure      404             {object}  shared.ErrorResponse
// @Router       /billing/invoices/{id} [get]
func (h *BillingHandler) GetInvoice(w http.ResponseWriter, r *http.Request) {
	workspaceID := middleware.GetWorkspaceID(r.Context())
	invoiceID := chi.URLParam(r, "id")

	invoice, err := h.Service.GetInvoice(r.Context(), workspaceID, invoiceID)
	if err != nil {
		shared.NotFound(w, r, "billing", "invoice not found")
		return
	}

	shared.OK(w, r, invoice)
}

// PayInvoice godoc
// @Summary      Pay invoice
// @Description  Initiate payment for an invoice via Xendit. Returns payment URL.
// @Tags         Billing
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        X-Workspace-ID  header    string                        true  "Workspace ID"
// @Param        body            body      billing.CreatePaymentRequest  true  "Payment data"
// @Success      201             {object}  shared.APIResponse{data=billing.PaymentResponse}
// @Failure      400             {object}  shared.ErrorResponse
// @Router       /billing/pay [post]
func (h *BillingHandler) PayInvoice(w http.ResponseWriter, r *http.Request) {
	workspaceID := middleware.GetWorkspaceID(r.Context())

	var req billing.CreatePaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.BadRequest(w, r, "billing", "invalid request body")
		return
	}

	if req.InvoiceID == "" {
		shared.BadRequest(w, r, "billing", "invoice_id is required")
		return
	}

	payment, err := h.Service.PayInvoice(r.Context(), workspaceID, req.InvoiceID)
	if err != nil {
		shared.BadRequest(w, r, "billing", err.Error())
		return
	}

	shared.Created(w, r, payment)
}

// ListPayments godoc
// @Summary      List payments
// @Description  Returns all payment attempts for the current workspace.
// @Tags         Billing
// @Produce      json
// @Security     BearerAuth
// @Param        X-Workspace-ID  header    string  true   "Workspace ID"
// @Param        limit           query     int     false  "Limit (default 20)"
// @Param        offset          query     int     false  "Offset (default 0)"
// @Success      200             {object}  shared.APIResponse{data=[]billing.PaymentResponse}
// @Failure      401             {object}  shared.ErrorResponse
// @Router       /billing/payments [get]
func (h *BillingHandler) ListPayments(w http.ResponseWriter, r *http.Request) {
	workspaceID := middleware.GetWorkspaceID(r.Context())

	limit, offset := 20, 0
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	payments, err := h.Service.ListPayments(r.Context(), workspaceID, limit, offset)
	if err != nil {
		shared.InternalError(w, r, "billing", "failed to list payments")
		return
	}

	shared.OK(w, r, payments)
}

// XenditWebhook godoc
// @Summary      Xendit payment webhook
// @Description  Receives payment status callbacks from Xendit. Verified by x-callback-token header.
// @Tags         Webhooks
// @Accept       json
// @Produce      json
// @Param        x-callback-token  header  string  true  "Xendit webhook verification token"
// @Success      200               {object}  shared.APIResponse{data=billing.MessageResponse}
// @Failure      400               {object}  shared.ErrorResponse
// @Failure      401               {object}  shared.ErrorResponse
// @Router       /webhooks/xendit [post]
func (h *BillingHandler) XenditWebhook(w http.ResponseWriter, r *http.Request) {
	// Verify webhook token
	callbackToken := r.Header.Get("x-callback-token")
	if !h.Service.Xendit.VerifyWebhookToken(callbackToken) {
		shared.Unauthorized(w, r, "webhook", "invalid callback token")
		return
	}

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		shared.BadRequest(w, r, "webhook", "failed to read body")
		return
	}

	// Parse payload
	var payload billing.XenditWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		shared.BadRequest(w, r, "webhook", "invalid payload")
		return
	}

	// Process webhook
	if err := h.Service.HandleWebhook(r.Context(), &payload); err != nil {
		shared.BadRequest(w, r, "webhook", err.Error())
		return
	}

	shared.OK(w, r, billing.MessageResponse{Message: "webhook processed"})
}

// AdminListInvoices godoc
// @Summary      List all invoices (admin)
// @Description  Returns all invoices across all workspaces. SUPER_ADMIN/ENGINEER only.
// @Tags         Admin Billing
// @Produce      json
// @Security     BearerAuth
// @Param        limit   query     int  false  "Limit (default 50)"
// @Param        offset  query     int  false  "Offset (default 0)"
// @Success      200     {object}  shared.APIResponse{data=[]billing.InvoiceResponse}
// @Failure      401     {object}  shared.ErrorResponse
// @Failure      403     {object}  shared.ErrorResponse
// @Router       /admin/billing/invoices [get]
func (h *BillingHandler) AdminListInvoices(w http.ResponseWriter, r *http.Request) {
	limit, offset := 50, 0
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	invoices, err := h.Service.ListAllInvoices(r.Context(), limit, offset)
	if err != nil {
		shared.InternalError(w, r, "billing", "failed to list invoices")
		return
	}

	shared.OK(w, r, invoices)
}
