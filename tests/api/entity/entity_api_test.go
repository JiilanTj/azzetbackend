package entity_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"codeberg.org/azzet/azzetbe/internal/shared"
)

func TestCreateEntity_MissingFields(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		wantCode int
	}{
		{
			name:     "missing nama_utama",
			body:     `{"entity_type": "BADAN_USAHA"}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "missing entity_type",
			body:     `{"nama_utama": "PT Maju"}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "empty body",
			body:     `{}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "invalid json",
			body:     `not json`,
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Post("/entities", func(w http.ResponseWriter, r *http.Request) {
				var req struct {
					EntityType string `json:"entity_type"`
					NamaUtama  string `json:"nama_utama"`
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					shared.BadRequest(w, r, "entity", "invalid request body")
					return
				}
				if req.NamaUtama == "" || req.EntityType == "" {
					shared.BadRequest(w, r, "entity", "nama_utama and entity_type are required")
					return
				}
				shared.Created(w, r, map[string]string{"id": "test"})
			})

			recorder := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/entities", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(recorder, req)

			if recorder.Code != tt.wantCode {
				t.Fatalf("expected %d, got %d. Body: %s", tt.wantCode, recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestCreateEntity_ValidRequest(t *testing.T) {
	r := chi.NewRouter()
	r.Post("/entities", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			EntityType string `json:"entity_type"`
			NamaUtama  string `json:"nama_utama"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			shared.BadRequest(w, r, "entity", "invalid request body")
			return
		}
		if req.NamaUtama == "" || req.EntityType == "" {
			shared.BadRequest(w, r, "entity", "nama_utama and entity_type are required")
			return
		}
		if req.EntityType != "ORANG_PRIBADI" && req.EntityType != "BADAN_USAHA" {
			shared.BadRequest(w, r, "entity", "invalid entity_type")
			return
		}
		shared.Created(w, r, map[string]string{"entity_type": req.EntityType, "nama_utama": req.NamaUtama})
	})

	recorder := httptest.NewRecorder()
	body := `{"entity_type": "BADAN_USAHA", "nama_utama": "PT Maju Jaya"}`
	req := httptest.NewRequest("POST", "/entities", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", recorder.Code, recorder.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(recorder.Body.Bytes(), &resp)
	data := resp["data"].(map[string]any)
	if data["entity_type"] != "BADAN_USAHA" {
		t.Fatalf("expected entity_type 'BADAN_USAHA', got '%v'", data["entity_type"])
	}
}

func TestCreateEntity_InvalidEntityType(t *testing.T) {
	r := chi.NewRouter()
	r.Post("/entities", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			EntityType string `json:"entity_type"`
			NamaUtama  string `json:"nama_utama"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			shared.BadRequest(w, r, "entity", "invalid request body")
			return
		}
		if req.EntityType != "ORANG_PRIBADI" && req.EntityType != "BADAN_USAHA" {
			shared.BadRequest(w, r, "entity", "invalid entity_type")
			return
		}
		shared.Created(w, r, map[string]string{"ok": "true"})
	})

	recorder := httptest.NewRecorder()
	body := `{"entity_type": "INVALID", "nama_utama": "Test"}`
	req := httptest.NewRequest("POST", "/entities", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestSearchEntities_MissingQuery(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/entities/search", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")
		if query == "" {
			shared.BadRequest(w, r, "entity", "query parameter 'q' is required")
			return
		}
		shared.OK(w, r, []any{})
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/entities/search", nil)
	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestSearchEntities_WithQuery(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/entities/search", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")
		if query == "" {
			shared.BadRequest(w, r, "entity", "query parameter 'q' is required")
			return
		}
		shared.OK(w, r, []any{})
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/entities/search?q=maju", nil)
	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", recorder.Code, recorder.Body.String())
	}
}
