package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"codeberg.org/azzet/azzetbe/internal/shared"
)

const WorkspaceIDKey contextKey = "workspace_id"
const WorkspaceRoleKey contextKey = "workspace_role"
const WorkspacePermissionsKey contextKey = "workspace_permissions"

type WorkspaceMiddleware struct {
	VerifyAccess func(ctx context.Context, workspaceID, userID string) (role string, permissions []byte, err error)
}

func NewWorkspaceMiddleware(verifyAccess func(ctx context.Context, workspaceID, userID string) (string, []byte, error)) *WorkspaceMiddleware {
	return &WorkspaceMiddleware{VerifyAccess: verifyAccess}
}

// RequireWorkspace extracts X-Workspace-ID header and verifies access
func (m *WorkspaceMiddleware) RequireWorkspace(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		workspaceID := r.Header.Get("X-Workspace-ID")
		if workspaceID == "" {
			shared.BadRequest(w, r, "workspace", "X-Workspace-ID header is required")
			return
		}

		userID := GetUserID(r.Context())
		if userID == "" {
			shared.Unauthorized(w, r, "workspace", "unauthorized")
			return
		}

		role, permissions, err := m.VerifyAccess(r.Context(), workspaceID, userID)
		if err != nil {
			shared.Forbidden(w, r, "workspace", "you don't have access to this workspace")
			return
		}

		ctx := context.WithValue(r.Context(), WorkspaceIDKey, workspaceID)
		ctx = context.WithValue(ctx, WorkspaceRoleKey, role)
		ctx = context.WithValue(ctx, WorkspacePermissionsKey, permissions)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequirePermission checks if the user has a specific permission in the workspace
func (m *WorkspaceMiddleware) RequirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			permissions := GetWorkspacePermissions(r.Context())
			if hasPermission(permissions, permission) {
				next.ServeHTTP(w, r)
				return
			}
			shared.Forbidden(w, r, "workspace", "insufficient permissions")
		})
	}
}

// GetWorkspaceID extracts workspace ID from context
func GetWorkspaceID(ctx context.Context) string {
	id, _ := ctx.Value(WorkspaceIDKey).(string)
	return id
}

// GetWorkspaceRole extracts workspace role from context
func GetWorkspaceRole(ctx context.Context) string {
	role, _ := ctx.Value(WorkspaceRoleKey).(string)
	return role
}

// GetWorkspacePermissions extracts workspace permissions from context
func GetWorkspacePermissions(ctx context.Context) []byte {
	perms, _ := ctx.Value(WorkspacePermissionsKey).([]byte)
	return perms
}

// RequireAssignedRole denies KARYAWAN members with no role permissions (default-deny).
func (m *WorkspaceMiddleware) RequireAssignedRole(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role := GetWorkspaceRole(r.Context())
		if role == "PEMILIK" {
			next.ServeHTTP(w, r)
			return
		}

		perms := GetWorkspacePermissions(r.Context())
		if len(perms) == 0 {
			shared.Forbidden(w, r, "workspace", "no role assigned")
			return
		}

		var permissions []string
		if err := json.Unmarshal(perms, &permissions); err != nil || len(permissions) == 0 {
			shared.Forbidden(w, r, "workspace", "no role assigned")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// hasPermission checks if permissions JSON array contains the required permission
func hasPermission(permissionsJSON []byte, required string) bool {
	if len(permissionsJSON) == 0 {
		return false
	}

	var permissions []string
	if err := json.Unmarshal(permissionsJSON, &permissions); err != nil {
		return false
	}

	for _, p := range permissions {
		// Wildcard: "*" means all permissions
		if p == "*" {
			return true
		}
		// Exact match
		if p == required {
			return true
		}
		// Resource wildcard: "transaction:*" matches "transaction:create"
		if strings.HasSuffix(p, ":*") {
			prefix := strings.TrimSuffix(p, "*")
			if strings.HasPrefix(required, prefix) {
				return true
			}
		}
	}

	return false
}
