package middleware

import (
	"context"
	"net/http"
	"strings"

	"codeberg.org/azzet/azzetbe/internal/shared"
)

type contextKey string

const UserIDKey contextKey = "user_id"

type AuthMiddleware struct {
	JWT           *shared.JWTService
	IsBlacklisted func(ctx context.Context, jti string) (bool, error)
}

func NewAuthMiddleware(jwt *shared.JWTService, isBlacklisted func(ctx context.Context, jti string) (bool, error)) *AuthMiddleware {
	return &AuthMiddleware{
		JWT:           jwt,
		IsBlacklisted: isBlacklisted,
	}
}

// Authenticate validates the JWT access token from Authorization header
func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			shared.Unauthorized(w, r, "auth", "missing authorization header")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			shared.Unauthorized(w, r, "auth", "invalid authorization header format")
			return
		}

		tokenString := parts[1]

		claims, err := m.JWT.ValidateAccessToken(tokenString)
		if err != nil {
			shared.Unauthorized(w, r, "auth", "invalid or expired token")
			return
		}

		// Check blacklist in Redis
		if m.IsBlacklisted != nil {
			blacklisted, err := m.IsBlacklisted(r.Context(), claims.JTI)
			if err != nil {
				shared.InternalError(w, r, "auth", "failed to verify token")
				return
			}
			if blacklisted {
				shared.Unauthorized(w, r, "auth", "token has been revoked")
				return
			}
		}

		// Set user ID in context
		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserID extracts user ID from request context
func GetUserID(ctx context.Context) string {
	userID, _ := ctx.Value(UserIDKey).(string)
	return userID
}

// GetAccessToken extracts the raw access token from Authorization header
func GetAccessToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 {
		return ""
	}
	return parts[1]
}

// GetRefreshToken extracts refresh token from HttpOnly cookie
func GetRefreshToken(r *http.Request) string {
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		return ""
	}
	return cookie.Value
}

// MaxBodySize limits the request body size
func MaxBodySize(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}

// GetClientIP extracts the real client IP from the request
func GetClientIP(r *http.Request) string {
	// Chi's RealIP middleware sets RemoteAddr
	ip := r.RemoteAddr
	// Strip port if present
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}
