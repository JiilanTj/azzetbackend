package middleware

import (
	"context"
	"net/http"
	"strings"

	"codeberg.org/azzet/azzetbe/internal/shared"
)

const AdminIDKey contextKey = "admin_id"
const AdminRoleKey contextKey = "admin_role"
const AdminScopeKey contextKey = "admin_scope"

type AdminMiddleware struct {
	JWT           *shared.JWTService
	IsBlacklisted func(ctx context.Context, jti string) (bool, error)
	GetAdminRole  func(ctx context.Context, adminID string) (string, error)
}

func NewAdminMiddleware(jwt *shared.JWTService, isBlacklisted func(ctx context.Context, jti string) (bool, error), getAdminRole func(ctx context.Context, adminID string) (string, error)) *AdminMiddleware {
	return &AdminMiddleware{
		JWT:           jwt,
		IsBlacklisted: isBlacklisted,
		GetAdminRole:  getAdminRole,
	}
}

// Authenticate validates admin JWT token
func (m *AdminMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			shared.Unauthorized(w, r, "admin", "missing authorization header")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			shared.Unauthorized(w, r, "admin", "invalid authorization header format")
			return
		}

		tokenString := parts[1]

		claims, err := m.JWT.ValidateAccessToken(tokenString)
		if err != nil {
			shared.Unauthorized(w, r, "admin", "invalid or expired token")
			return
		}

		// Check blacklist
		if m.IsBlacklisted != nil {
			blacklisted, err := m.IsBlacklisted(r.Context(), claims.JTI)
			if err != nil {
				shared.InternalError(w, r, "admin", "failed to verify token")
				return
			}
			if blacklisted {
				shared.Unauthorized(w, r, "admin", "token has been revoked")
				return
			}
		}

		// Get admin role and verify account is active
		role := ""
		if m.GetAdminRole != nil {
			role, err = m.GetAdminRole(r.Context(), claims.UserID)
			if err != nil {
				shared.Unauthorized(w, r, "admin", "admin not found")
				return
			}
		}

		ctx := context.WithValue(r.Context(), AdminIDKey, claims.UserID)
		ctx = context.WithValue(ctx, AdminRoleKey, role)
		ctx = context.WithValue(ctx, AdminScopeKey, claims.Scope)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireFullAuth rejects MFA-setup-scoped tokens on routes that need full admin access.
func (m *AdminMiddleware) RequireFullAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if GetAdminScope(r.Context()) == shared.TokenScopeMFASetup {
			shared.Forbidden(w, r, "admin", "MFA setup required before accessing this resource")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireMFASetupScope only allows tokens issued for MFA setup (blocks full admin tokens).
func (m *AdminMiddleware) RequireMFASetupScope(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if GetAdminScope(r.Context()) != shared.TokenScopeMFASetup {
			shared.Forbidden(w, r, "admin", "this endpoint requires an MFA setup token")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireRole checks if admin has the required role
func (m *AdminMiddleware) RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			adminRole := GetAdminRole(r.Context())
			if adminRole == "" {
				shared.Forbidden(w, r, "admin", "insufficient permissions")
				return
			}

			for _, role := range roles {
				if adminRole == role {
					next.ServeHTTP(w, r)
					return
				}
			}

			// SUPER_ADMIN always has access
			if adminRole == "SUPER_ADMIN" {
				next.ServeHTTP(w, r)
				return
			}

			shared.Forbidden(w, r, "admin", "insufficient permissions")
		})
	}
}

// GetAdminID extracts admin ID from context
func GetAdminID(ctx context.Context) string {
	id, _ := ctx.Value(AdminIDKey).(string)
	return id
}

// GetAdminRole extracts admin role from context
func GetAdminRole(ctx context.Context) string {
	role, _ := ctx.Value(AdminRoleKey).(string)
	return role
}

// GetAdminScope extracts token scope from context
func GetAdminScope(ctx context.Context) string {
	scope, _ := ctx.Value(AdminScopeKey).(string)
	return scope
}

// GetAdminIDOrError returns admin ID or false if missing from context
func GetAdminIDOrError(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(AdminIDKey).(string)
	return id, ok && id != ""
}
