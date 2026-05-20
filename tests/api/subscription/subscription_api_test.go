package subscription_test

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

func TestSubscribe_MissingPlanID(t *testing.T) {
	r := chi.NewRouter()
	r.Post("/subscription", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			PlanID string `json:"plan_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			shared.BadRequest(w, r, "subscription", "invalid request body")
			return
		}
		if req.PlanID == "" {
			shared.BadRequest(w, r, "subscription", "plan_id is required")
			return
		}
		shared.Created(w, r, map[string]string{"ok": "true"})
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/subscription", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestSubscribe_ValidRequest(t *testing.T) {
	r := chi.NewRouter()
	r.Post("/subscription", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			PlanID       string  `json:"plan_id"`
			BillingCycle *string `json:"billing_cycle"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			shared.BadRequest(w, r, "subscription", "invalid request body")
			return
		}
		if req.PlanID == "" {
			shared.BadRequest(w, r, "subscription", "plan_id is required")
			return
		}
		shared.Created(w, r, map[string]string{"plan_id": req.PlanID, "status": "active"})
	})

	planID := uuid.New().String()
	recorder := httptest.NewRecorder()
	body := `{"plan_id": "` + planID + `", "billing_cycle": "monthly"}`
	req := httptest.NewRequest("POST", "/subscription", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", recorder.Code, recorder.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(recorder.Body.Bytes(), &resp)
	data := resp["data"].(map[string]any)
	if data["plan_id"] != planID {
		t.Fatalf("expected plan_id '%s', got '%v'", planID, data["plan_id"])
	}
}

func TestChangePlan_MissingPlanID(t *testing.T) {
	r := chi.NewRouter()
	r.Post("/subscription/change", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			PlanID string `json:"plan_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			shared.BadRequest(w, r, "subscription", "invalid request body")
			return
		}
		if req.PlanID == "" {
			shared.BadRequest(w, r, "subscription", "plan_id is required")
			return
		}
		shared.OK(w, r, map[string]string{"ok": "true"})
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/subscription/change", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestSubscriptionEndpoints_RequireWorkspaceHeader(t *testing.T) {
	verifyAccess := func(ctx context.Context, workspaceID, userID string) (string, []byte, error) {
		return "PEMILIK", []byte(`["*"]`), nil
	}
	wsMw := middleware.NewWorkspaceMiddleware(verifyAccess)

	endpoints := []struct {
		method string
		path   string
	}{
		{"POST", "/subscription"},
		{"GET", "/subscription"},
		{"GET", "/subscription/history"},
		{"POST", "/subscription/cancel"},
		{"POST", "/subscription/change"},
		{"GET", "/subscription/usage"},
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

func TestSubscriptionEndpoints_WithWorkspaceHeader(t *testing.T) {
	verifyAccess := func(ctx context.Context, workspaceID, userID string) (string, []byte, error) {
		return "PEMILIK", []byte(`["*"]`), nil
	}
	wsMw := middleware.NewWorkspaceMiddleware(verifyAccess)

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), middleware.UserIDKey, "user-123")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.With(wsMw.RequireWorkspace).Get("/subscription", func(w http.ResponseWriter, r *http.Request) {
		wsID := middleware.GetWorkspaceID(r.Context())
		shared.OK(w, r, map[string]string{"workspace_id": wsID})
	})

	wsID := uuid.New().String()
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/subscription", nil)
	req.Header.Set("X-Workspace-ID", wsID)
	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", recorder.Code, recorder.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(recorder.Body.Bytes(), &resp)
	data := resp["data"].(map[string]any)
	if data["workspace_id"] != wsID {
		t.Fatalf("expected workspace_id '%s', got '%v'", wsID, data["workspace_id"])
	}
}
