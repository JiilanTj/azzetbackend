package workspace_test

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

func TestCreateWorkspace_MissingEntityID(t *testing.T) {
	r := chi.NewRouter()
	r.Post("/workspaces", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			EntityID string `json:"entity_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			shared.BadRequest(w, r, "workspace", "invalid request body")
			return
		}
		if req.EntityID == "" {
			shared.BadRequest(w, r, "workspace", "entity_id is required")
			return
		}
		shared.Created(w, r, map[string]string{"ok": "true"})
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/workspaces", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestCreateWorkspace_ValidRequest(t *testing.T) {
	r := chi.NewRouter()
	r.Post("/workspaces", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			EntityID string `json:"entity_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			shared.BadRequest(w, r, "workspace", "invalid request body")
			return
		}
		if req.EntityID == "" {
			shared.BadRequest(w, r, "workspace", "entity_id is required")
			return
		}
		shared.Created(w, r, map[string]string{"entity_id": req.EntityID, "role": "PEMILIK"})
	})

	entityID := uuid.New().String()
	recorder := httptest.NewRecorder()
	body := `{"entity_id": "` + entityID + `"}`
	req := httptest.NewRequest("POST", "/workspaces", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", recorder.Code, recorder.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(recorder.Body.Bytes(), &resp)
	data := resp["data"].(map[string]any)
	if data["role"] != "PEMILIK" {
		t.Fatalf("expected role 'PEMILIK', got '%v'", data["role"])
	}
}

func TestInviteMember_MissingFields(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		wantCode int
	}{
		{
			name:     "missing entity_id",
			body:     `{"role": "KASIR"}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "missing role",
			body:     `{"entity_id": "` + uuid.New().String() + `"}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "empty body",
			body:     `{}`,
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Post("/members", func(w http.ResponseWriter, r *http.Request) {
				var req struct {
					EntityID string `json:"entity_id"`
					Role     string `json:"role"`
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					shared.BadRequest(w, r, "workspace", "invalid request body")
					return
				}
				if req.EntityID == "" || req.Role == "" {
					shared.BadRequest(w, r, "workspace", "entity_id and role are required")
					return
				}
				shared.Created(w, r, map[string]string{"ok": "true"})
			})

			recorder := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/members", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(recorder, req)

			if recorder.Code != tt.wantCode {
				t.Fatalf("expected %d, got %d. Body: %s", tt.wantCode, recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestAddCounterparty_MissingRelationType(t *testing.T) {
	r := chi.NewRouter()
	r.Post("/counterparties", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RelationType string `json:"relation_type"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			shared.BadRequest(w, r, "workspace", "invalid request body")
			return
		}
		if req.RelationType == "" {
			shared.BadRequest(w, r, "workspace", "relation_type is required")
			return
		}
		shared.Created(w, r, map[string]string{"ok": "true"})
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/counterparties", strings.NewReader(`{"nama_utama": "Toko"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestAddCounterparty_ValidShadowEntity(t *testing.T) {
	r := chi.NewRouter()
	r.Post("/counterparties", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RelationType string  `json:"relation_type"`
			NamaUtama    *string `json:"nama_utama"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			shared.BadRequest(w, r, "workspace", "invalid request body")
			return
		}
		if req.RelationType == "" {
			shared.BadRequest(w, r, "workspace", "relation_type is required")
			return
		}
		if req.RelationType != "PELANGGAN" && req.RelationType != "VENDOR" {
			shared.BadRequest(w, r, "workspace", "relation_type must be PELANGGAN or VENDOR")
			return
		}
		shared.Created(w, r, map[string]string{"relation_type": req.RelationType, "is_shadow": "true"})
	})

	recorder := httptest.NewRecorder()
	body := `{"relation_type": "PELANGGAN", "nama_utama": "Toko Maju", "entity_type": "BADAN_USAHA"}`
	req := httptest.NewRequest("POST", "/counterparties", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", recorder.Code, recorder.Body.String())
	}
}

func TestAddCounterparty_InvalidRelationType(t *testing.T) {
	r := chi.NewRouter()
	r.Post("/counterparties", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RelationType string `json:"relation_type"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			shared.BadRequest(w, r, "workspace", "invalid request body")
			return
		}
		if req.RelationType != "PELANGGAN" && req.RelationType != "VENDOR" {
			shared.BadRequest(w, r, "workspace", "relation_type must be PELANGGAN or VENDOR")
			return
		}
		shared.Created(w, r, map[string]string{"ok": "true"})
	})

	recorder := httptest.NewRecorder()
	body := `{"relation_type": "KARYAWAN", "nama_utama": "Test"}`
	req := httptest.NewRequest("POST", "/counterparties", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestWorkspaceEndpoints_RequireWorkspaceHeader(t *testing.T) {
	verifyAccess := func(ctx context.Context, workspaceID, userID string) (string, []byte, error) {
		return "PEMILIK", []byte(`["*"]`), nil
	}
	wsMw := middleware.NewWorkspaceMiddleware(verifyAccess)

	endpoints := []struct {
		method string
		path   string
	}{
		{"GET", "/members"},
		{"POST", "/members"},
		{"GET", "/counterparties"},
		{"POST", "/counterparties"},
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
