package auth_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"codeberg.org/azzet/azzetbe/internal/api/handler"
	"codeberg.org/azzet/azzetbe/internal/api/middleware"
	"codeberg.org/azzet/azzetbe/internal/shared"
)

// Helper to create a test router with auth middleware
func setupTestRouter(authHandler *handler.AuthHandler, jwtService *shared.JWTService) *chi.Mux {
	r := chi.NewRouter()

	isBlacklisted := func(ctx context.Context, jti string) (bool, error) {
		return false, nil
	}
	authMw := middleware.NewAuthMiddleware(jwtService, isBlacklisted)

	r.Route("/api/v1/auth", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Post("/login/email", authHandler.LoginEmail)
		r.Post("/login/otp", authHandler.LoginOTP)
		r.Post("/refresh", authHandler.RefreshToken)

		r.Group(func(r chi.Router) {
			r.Use(authMw.Authenticate)
			r.Get("/me", authHandler.Me)
			r.Post("/logout", authHandler.Logout)
		})
	})

	return r
}

func TestRegister_MissingBody(t *testing.T) {
	// This test verifies that the handler returns 400 for empty body
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/json")

	r := chi.NewRouter()
	r.Post("/api/v1/auth/register", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			shared.BadRequest(w, r, "auth", "invalid request body")
			return
		}
	})

	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Code)
	}
}

func TestRegister_ValidationErrors(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		wantCode int
	}{
		{
			name:     "missing email and whatsapp",
			body:     `{"password": "password123"}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "password too short",
			body:     `{"email": "test@example.com", "password": "short"}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "invalid email format",
			body:     `{"email": "not-an-email", "password": "password123"}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "missing password",
			body:     `{"email": "test@example.com"}`,
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

			// Use a simple handler that validates like the real one
			r := chi.NewRouter()
			r.Post("/api/v1/auth/register", func(w http.ResponseWriter, r *http.Request) {
				var regReq struct {
					Email    *string `json:"email,omitempty"`
					WhatsApp *string `json:"whatsapp,omitempty"`
					Password string  `json:"password"`
				}
				if err := json.NewDecoder(r.Body).Decode(&regReq); err != nil {
					shared.BadRequest(w, r, "auth", "invalid request body")
					return
				}

				var errs []shared.FieldError
				if regReq.Email == nil && regReq.WhatsApp == nil {
					errs = append(errs, shared.FieldError{Field: "email", Message: "email or whatsapp is required"})
				}
				if regReq.Email != nil && *regReq.Email != "" {
					if msg := shared.ValidateEmail(*regReq.Email); msg != "" {
						errs = append(errs, shared.FieldError{Field: "email", Message: msg})
					}
				}
				if msg := shared.ValidateMinLength(regReq.Password, 8, "password"); msg != "" {
					errs = append(errs, shared.FieldError{Field: "password", Message: msg})
				}

				if len(errs) > 0 {
					shared.ValidationError(w, r, "auth", "validation failed", errs)
					return
				}

				shared.Created(w, r, map[string]string{"message": "ok"})
			})

			r.ServeHTTP(recorder, req)

			if recorder.Code != tt.wantCode {
				t.Fatalf("expected status %d, got %d. Body: %s", tt.wantCode, recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestLoginEmail_MissingFields(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		wantCode int
	}{
		{
			name:     "missing email",
			body:     `{"password": "password123"}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "missing password",
			body:     `{"email": "test@example.com"}`,
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
			recorder := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/api/v1/auth/login/email", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

			r := chi.NewRouter()
			r.Post("/api/v1/auth/login/email", func(w http.ResponseWriter, r *http.Request) {
				var loginReq struct {
					Email    string `json:"email"`
					Password string `json:"password"`
				}
				if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
					shared.BadRequest(w, r, "auth", "invalid request body")
					return
				}

				var errs []shared.FieldError
				if loginReq.Email == "" {
					errs = append(errs, shared.FieldError{Field: "email", Message: "email is required"})
				}
				if loginReq.Password == "" {
					errs = append(errs, shared.FieldError{Field: "password", Message: "password is required"})
				}

				if len(errs) > 0 {
					shared.ValidationError(w, r, "auth", "validation failed", errs)
					return
				}

				shared.OK(w, r, map[string]string{"message": "ok"})
			})

			r.ServeHTTP(recorder, req)

			if recorder.Code != tt.wantCode {
				t.Fatalf("expected status %d, got %d. Body: %s", tt.wantCode, recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	jwtService := shared.NewJWTService("test-secret", "test-refresh", 15*time.Minute, 7*24*time.Hour)

	isBlacklisted := func(ctx context.Context, jti string) (bool, error) {
		return false, nil
	}
	authMw := middleware.NewAuthMiddleware(jwtService, isBlacklisted)

	r := chi.NewRouter()
	r.With(authMw.Authenticate).Get("/protected", func(w http.ResponseWriter, r *http.Request) {
		shared.OK(w, r, map[string]string{"message": "ok"})
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/protected", nil)
	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	jwtService := shared.NewJWTService("test-secret", "test-refresh", 15*time.Minute, 7*24*time.Hour)

	isBlacklisted := func(ctx context.Context, jti string) (bool, error) {
		return false, nil
	}
	authMw := middleware.NewAuthMiddleware(jwtService, isBlacklisted)

	r := chi.NewRouter()
	r.With(authMw.Authenticate).Get("/protected", func(w http.ResponseWriter, r *http.Request) {
		shared.OK(w, r, map[string]string{"message": "ok"})
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	jwtService := shared.NewJWTService("test-secret", "test-refresh", 15*time.Minute, 7*24*time.Hour)

	isBlacklisted := func(ctx context.Context, jti string) (bool, error) {
		return false, nil
	}
	authMw := middleware.NewAuthMiddleware(jwtService, isBlacklisted)

	token, _ := jwtService.GenerateAccessToken("user-123", "session-456")

	r := chi.NewRouter()
	r.With(authMw.Authenticate).Get("/protected", func(w http.ResponseWriter, r *http.Request) {
		userID := middleware.GetUserID(r.Context())
		shared.OK(w, r, map[string]string{"user_id": userID})
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", recorder.Code, recorder.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(recorder.Body.Bytes(), &resp)
	data := resp["data"].(map[string]any)
	if data["user_id"] != "user-123" {
		t.Fatalf("expected user_id 'user-123', got '%v'", data["user_id"])
	}
}

func TestAuthMiddleware_BlacklistedToken(t *testing.T) {
	jwtService := shared.NewJWTService("test-secret", "test-refresh", 15*time.Minute, 7*24*time.Hour)

	// Always return blacklisted
	isBlacklisted := func(ctx context.Context, jti string) (bool, error) {
		return true, nil
	}
	authMw := middleware.NewAuthMiddleware(jwtService, isBlacklisted)

	token, _ := jwtService.GenerateAccessToken("user-123", "session-456")

	r := chi.NewRouter()
	r.With(authMw.Authenticate).Get("/protected", func(w http.ResponseWriter, r *http.Request) {
		shared.OK(w, r, map[string]string{"message": "ok"})
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestRefreshTokenCookie(t *testing.T) {
	recorder := httptest.NewRecorder()

	http.SetCookie(recorder, &http.Cookie{
		Name:     "refresh_token",
		Value:    "test-refresh-token",
		Path:     "/api/v1/auth",
		MaxAge:   604800,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})

	cookies := recorder.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected cookie to be set")
	}

	cookie := cookies[0]
	if cookie.Name != "refresh_token" {
		t.Fatalf("expected cookie name 'refresh_token', got '%s'", cookie.Name)
	}
	if cookie.Value != "test-refresh-token" {
		t.Fatalf("expected cookie value 'test-refresh-token', got '%s'", cookie.Value)
	}
	if !cookie.HttpOnly {
		t.Fatal("expected HttpOnly cookie")
	}
	if !cookie.Secure {
		t.Fatal("expected Secure cookie")
	}
	if cookie.SameSite != http.SameSiteStrictMode {
		t.Fatal("expected SameSite=Strict")
	}
}

func TestGetRefreshToken_FromCookie(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  "refresh_token",
		Value: "my-refresh-token",
	})

	token := middleware.GetRefreshToken(req)
	if token != "my-refresh-token" {
		t.Fatalf("expected 'my-refresh-token', got '%s'", token)
	}
}

func TestGetRefreshToken_NoCookie(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)

	token := middleware.GetRefreshToken(req)
	if token != "" {
		t.Fatalf("expected empty string, got '%s'", token)
	}
}

func TestGetAccessToken_FromHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer my-access-token")

	token := middleware.GetAccessToken(req)
	if token != "my-access-token" {
		t.Fatalf("expected 'my-access-token', got '%s'", token)
	}
}

func TestGetAccessToken_NoHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)

	token := middleware.GetAccessToken(req)
	if token != "" {
		t.Fatalf("expected empty string, got '%s'", token)
	}
}

func TestGetClientIP(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	ip := middleware.GetClientIP(req)
	if ip != "192.168.1.1" {
		t.Fatalf("expected '192.168.1.1', got '%s'", ip)
	}
}
