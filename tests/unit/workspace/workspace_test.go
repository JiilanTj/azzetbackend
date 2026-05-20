package workspace_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"codeberg.org/azzet/azzetbe/internal/api/middleware"
	"codeberg.org/azzet/azzetbe/internal/db"
	"codeberg.org/azzet/azzetbe/internal/shared"
	"codeberg.org/azzet/azzetbe/internal/workspace"
)

func TestRelationToMemberResponse(t *testing.T) {
	now := time.Now()
	roleName := "KASIR"

	rel := &db.EntityRelation{
		ID:           uuid.New(),
		ObjectID:     uuid.New(),
		SubjectID:    uuid.New(),
		RelationType: workspace.RelationKaryawan,
		CustomAlias:  pgtype.Text{String: "Andi Accounting", Valid: true},
		Status:       "ACTIVE",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	entity := &db.Entity{
		ID:         rel.SubjectID,
		EntityType: "ORANG_PRIBADI",
		NamaUtama:  "Andi",
	}

	resp := workspace.RelationToMemberResponse(rel, entity, &roleName)

	if resp.ID != rel.ID.String() {
		t.Fatalf("expected ID '%s', got '%s'", rel.ID.String(), resp.ID)
	}
	if resp.EntityID != rel.SubjectID.String() {
		t.Fatalf("expected entity_id '%s', got '%s'", rel.SubjectID.String(), resp.EntityID)
	}
	if resp.EntityName != "Andi" {
		t.Fatalf("expected entity_name 'Andi', got '%s'", resp.EntityName)
	}
	if resp.RelationType != workspace.RelationKaryawan {
		t.Fatalf("expected relation_type '%s', got '%s'", workspace.RelationKaryawan, resp.RelationType)
	}
	if resp.CustomAlias == nil || *resp.CustomAlias != "Andi Accounting" {
		t.Fatalf("expected custom_alias 'Andi Accounting', got %v", resp.CustomAlias)
	}
	if resp.Role == nil || *resp.Role != "KASIR" {
		t.Fatalf("expected role 'KASIR', got %v", resp.Role)
	}
	if resp.Status != "ACTIVE" {
		t.Fatalf("expected status 'ACTIVE', got '%s'", resp.Status)
	}
}

func TestRelationToCounterpartyResponse(t *testing.T) {
	now := time.Now()

	rel := &db.EntityRelation{
		ID:           uuid.New(),
		ObjectID:     uuid.New(),
		SubjectID:    uuid.New(),
		RelationType: workspace.RelationPelanggan,
		CustomAlias:  pgtype.Text{String: "Toko Maju", Valid: true},
		Status:       "ACTIVE",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	entity := &db.Entity{
		ID:         rel.SubjectID,
		EntityType: "BADAN_USAHA",
		NamaUtama:  "PT Maju Jaya",
		IsShadow:   true,
	}

	resp := workspace.RelationToCounterpartyResponse(rel, entity)

	if resp.EntityName != "PT Maju Jaya" {
		t.Fatalf("expected entity_name 'PT Maju Jaya', got '%s'", resp.EntityName)
	}
	if resp.RelationType != workspace.RelationPelanggan {
		t.Fatalf("expected relation_type '%s', got '%s'", workspace.RelationPelanggan, resp.RelationType)
	}
	if !resp.IsShadow {
		t.Fatal("expected is_shadow true")
	}
	if resp.CustomAlias == nil || *resp.CustomAlias != "Toko Maju" {
		t.Fatalf("expected custom_alias 'Toko Maju', got %v", resp.CustomAlias)
	}
}

func TestRelationToMemberResponse_NilAlias(t *testing.T) {
	now := time.Now()

	rel := &db.EntityRelation{
		ID:           uuid.New(),
		ObjectID:     uuid.New(),
		SubjectID:    uuid.New(),
		RelationType: workspace.RelationPemilik,
		CustomAlias:  pgtype.Text{Valid: false},
		Status:       "ACTIVE",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	entity := &db.Entity{
		ID:         rel.SubjectID,
		EntityType: "ORANG_PRIBADI",
		NamaUtama:  "Owner",
	}

	resp := workspace.RelationToMemberResponse(rel, entity, nil)

	if resp.CustomAlias != nil {
		t.Fatalf("expected nil custom_alias, got %v", resp.CustomAlias)
	}
	if resp.Role != nil {
		t.Fatalf("expected nil role, got %v", resp.Role)
	}
}

func TestWorkspaceConstants(t *testing.T) {
	if workspace.RelationPemilik != "PEMILIK" {
		t.Fatalf("expected 'PEMILIK', got '%s'", workspace.RelationPemilik)
	}
	if workspace.RelationKaryawan != "KARYAWAN" {
		t.Fatalf("expected 'KARYAWAN', got '%s'", workspace.RelationKaryawan)
	}
	if workspace.RelationPelanggan != "PELANGGAN" {
		t.Fatalf("expected 'PELANGGAN', got '%s'", workspace.RelationPelanggan)
	}
	if workspace.RelationVendor != "VENDOR" {
		t.Fatalf("expected 'VENDOR', got '%s'", workspace.RelationVendor)
	}
}

func TestWorkspaceMiddleware_MissingHeader(t *testing.T) {
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
	r.With(wsMw.RequireWorkspace).Get("/test", func(w http.ResponseWriter, r *http.Request) {
		shared.OK(w, r, map[string]string{"ok": "true"})
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestWorkspaceMiddleware_ValidHeader(t *testing.T) {
	wsID := uuid.New().String()

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
	r.With(wsMw.RequireWorkspace).Get("/test", func(w http.ResponseWriter, r *http.Request) {
		id := middleware.GetWorkspaceID(r.Context())
		role := middleware.GetWorkspaceRole(r.Context())
		shared.OK(w, r, map[string]string{"workspace_id": id, "role": role})
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
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
	if data["role"] != "PEMILIK" {
		t.Fatalf("expected role 'PEMILIK', got '%v'", data["role"])
	}
}

func TestWorkspaceMiddleware_PermissionCheck(t *testing.T) {
	verifyAccess := func(ctx context.Context, workspaceID, userID string) (string, []byte, error) {
		return "KASIR", []byte(`["transaction:create","transaction:read","item:read"]`), nil
	}
	wsMw := middleware.NewWorkspaceMiddleware(verifyAccess)

	tests := []struct {
		name       string
		permission string
		wantCode   int
	}{
		{"allowed - exact match", "transaction:create", http.StatusOK},
		{"allowed - read", "transaction:read", http.StatusOK},
		{"denied - no permission", "report:read", http.StatusForbidden},
		{"denied - write not allowed", "transaction:delete", http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Use(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					ctx := context.WithValue(r.Context(), middleware.UserIDKey, "user-123")
					next.ServeHTTP(w, r.WithContext(ctx))
				})
			})
			r.With(wsMw.RequireWorkspace, wsMw.RequirePermission(tt.permission)).Get("/test", func(w http.ResponseWriter, r *http.Request) {
				shared.OK(w, r, map[string]string{"ok": "true"})
			})

			recorder := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("X-Workspace-ID", uuid.New().String())
			r.ServeHTTP(recorder, req)

			if recorder.Code != tt.wantCode {
				t.Fatalf("expected %d, got %d. Body: %s", tt.wantCode, recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestWorkspaceMiddleware_WildcardPermission(t *testing.T) {
	// PEMILIK has ["*"] = all permissions
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
	r.With(wsMw.RequireWorkspace, wsMw.RequirePermission("anything:here")).Get("/test", func(w http.ResponseWriter, r *http.Request) {
		shared.OK(w, r, map[string]string{"ok": "true"})
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Workspace-ID", uuid.New().String())
	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200 for wildcard permission, got %d", recorder.Code)
	}
}

func TestWorkspaceMiddleware_ResourceWildcard(t *testing.T) {
	// AKUNTAN has ["transaction:*"] = all transaction permissions
	verifyAccess := func(ctx context.Context, workspaceID, userID string) (string, []byte, error) {
		return "AKUNTAN", []byte(`["transaction:*","report:*"]`), nil
	}
	wsMw := middleware.NewWorkspaceMiddleware(verifyAccess)

	tests := []struct {
		name       string
		permission string
		wantCode   int
	}{
		{"transaction:create allowed", "transaction:create", http.StatusOK},
		{"transaction:delete allowed", "transaction:delete", http.StatusOK},
		{"report:read allowed", "report:read", http.StatusOK},
		{"item:create denied", "item:create", http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Use(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					ctx := context.WithValue(r.Context(), middleware.UserIDKey, "user-123")
					next.ServeHTTP(w, r.WithContext(ctx))
				})
			})
			r.With(wsMw.RequireWorkspace, wsMw.RequirePermission(tt.permission)).Get("/test", func(w http.ResponseWriter, r *http.Request) {
				shared.OK(w, r, map[string]string{"ok": "true"})
			})

			recorder := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("X-Workspace-ID", uuid.New().String())
			r.ServeHTTP(recorder, req)

			if recorder.Code != tt.wantCode {
				t.Fatalf("expected %d, got %d", tt.wantCode, recorder.Code)
			}
		})
	}
}
