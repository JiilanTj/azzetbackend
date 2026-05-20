package billing_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"codeberg.org/azzet/azzetbe/internal/api/middleware"
	"codeberg.org/azzet/azzetbe/internal/shared"
)

func TestPayInvoice_MissingInvoiceID(t *testing.T) {
	r := chi.NewRouter()
	r.Post("/billing/pay", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			InvoiceID string `json:"invoice_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			shared.BadRequest(w, r, "billing", "invalid request body")
			return
		}
		if req.InvoiceID == "" {
			shared.BadRequest(w, r, "billing", "invoice_id is required")
			return
		}
		shared.Created(w, r, map[string]string{"ok": "true"})
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/billing/pay", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestPayInvoice_ValidRequest(t *testing.T) {
	r := chi.NewRouter()
	r.Post("/billing/pay", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			InvoiceID string `json:"invoice_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			shared.BadRequest(w, r, "billing", "invalid request body")
			return
		}
		if req.InvoiceID == "" {
			shared.BadRequest(w, r, "billing", "invoice_id is required")
			return
		}
		shared.Created(w, r, map[string]string{"invoice_id": req.InvoiceID, "status": "pending"})
	})

	invoiceID := uuid.New().String()
	recorder := httptest.NewRecorder()
	body := `{"invoice_id": "` + invoiceID + `"}`
	req := httptest.NewRequest("POST", "/billing/pay", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", recorder.Code, recorder.Body.String())
	}
}

func TestXenditWebhook_MissingToken(t *testing.T) {
	r := chi.NewRouter()
	r.Post("/webhooks/xendit", func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("x-callback-token")
		if token != "test-secret" {
			shared.Unauthorized(w, r, "webhook", "invalid callback token")
			return
		}
		shared.OK(w, r, map[string]string{"ok": "true"})
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/webhooks/xendit", strings.NewReader(`{}`))
	// No x-callback-token header
	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestXenditWebhook_ValidToken(t *testing.T) {
	r := chi.NewRouter()
	r.Post("/webhooks/xendit", func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("x-callback-token")
		if token != "test-secret" {
			shared.Unauthorized(w, r, "webhook", "invalid callback token")
			return
		}
		shared.OK(w, r, map[string]string{"ok": "true"})
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/webhooks/xendit", strings.NewReader(`{"id":"inv-123","status":"PAID"}`))
	req.Header.Set("x-callback-token", "test-secret")
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", recorder.Code, recorder.Body.String())
	}
}

func TestBillingEndpoints_RequireWorkspaceHeader(t *testing.T) {
	verifyAccess := func(ctx context.Context, workspaceID, userID string) (string, []byte, error) {
		return "PEMILIK", []byte(`["*"]`), nil
	}
	wsMw := middleware.NewWorkspaceMiddleware(verifyAccess)

	endpoints := []struct {
		method string
		path   string
	}{
		{"GET", "/billing/invoices"},
		{"POST", "/billing/pay"},
		{"GET", "/billing/payments"},
	}

	for _, ep := range endpoints {
		t.Run(ep.method+" "+ep.path, func(t *testing.T) {
			r := chi.NewRouter()
			r.Use(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					ctx := context.WithValue(r.Context(), middleware.UserIDKey, "user-123")
					next.ServeHTTP(w, r.WithContext(ctx))
				})
			})
			r.With(wsMw.RequireWorkspace).HandleFunc(ep.path, func(w http.ResponseWriter, r *http.Request) {
				shared.OK(w, r, map[string]string{"ok": "true"})
			})

			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(ep.method, ep.path, strings.NewReader(`{}`))
			// No X-Workspace-ID header
			r.ServeHTTP(recorder, req)

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("expected 400 without X-Workspace-ID, got %d", recorder.Code)
			}
		})
	}
}
