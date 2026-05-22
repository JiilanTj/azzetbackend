package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"codeberg.org/azzet/azzetbe/internal/api/middleware"
	"codeberg.org/azzet/azzetbe/internal/auth"
	"codeberg.org/azzet/azzetbe/internal/shared"
)

type AuthHandler struct {
	Service            *auth.Service
	RefreshTokenExpiry time.Duration
	SecureCookie       bool
}

func NewAuthHandler(service *auth.Service, refreshTokenExpiry time.Duration, secureCookie bool) *AuthHandler {
	return &AuthHandler{
		Service:            service,
		RefreshTokenExpiry: refreshTokenExpiry,
		SecureCookie:       secureCookie,
	}
}

// Register godoc
// @Summary      Register a new user
// @Description  Create a new account with email or whatsapp. Password is always required as fallback.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      auth.RegisterRequest  true  "Registration data"
// @Success      201   {object}  shared.APIResponse{data=auth.RegisterResponse}
// @Failure      400   {object}  shared.ErrorResponse
// @Router       /auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req auth.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.BadRequest(w, r, "auth", "invalid request body")
		return
	}

	if errs := validateRegister(&req); len(errs) > 0 {
		shared.ValidationError(w, r, "auth", "validation failed", errs)
		return
	}

	user, err := h.Service.Register(r.Context(), &req)
	if err != nil {
		shared.BadRequest(w, r, "auth", err.Error())
		return
	}

	shared.Created(w, r, map[string]any{
		"user":    auth.UserToResponse(user),
		"message": "Registration successful. Please verify your account.",
	})
}

// LoginEmail godoc
// @Summary      Login with email and password
// @Description  Authenticate using email + password. Returns access token in body and refresh token in HttpOnly cookie.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      auth.LoginEmailRequest  true  "Login credentials"
// @Success      200   {object}  shared.APIResponse{data=auth.AuthResponse}
// @Failure      400   {object}  shared.ErrorResponse
// @Failure      401   {object}  shared.ErrorResponse
// @Header       200   {string}  Set-Cookie  "refresh_token=<token>; Path=/api/v1/auth; HttpOnly; Secure; SameSite=Strict"
// @Router       /auth/login/email [post]
func (h *AuthHandler) LoginEmail(w http.ResponseWriter, r *http.Request) {
	var req auth.LoginEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.BadRequest(w, r, "auth", "invalid request body")
		return
	}

	if errs := validateLoginEmail(&req); len(errs) > 0 {
		shared.ValidationError(w, r, "auth", "validation failed", errs)
		return
	}

	lc := loginContextFromRequest(r)
	resp, refreshToken, err := h.Service.LoginWithEmail(r.Context(), &req, lc)
	if err != nil {
		shared.Unauthorized(w, r, "auth", err.Error())
		return
	}

	h.setRefreshTokenCookie(w, refreshToken)
	shared.OK(w, r, resp)
}

// LoginOTP godoc
// @Summary      Login with WhatsApp OTP
// @Description  Authenticate using WhatsApp number + OTP code. Request OTP first via /auth/otp/request.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      auth.LoginOTPRequest  true  "WhatsApp + OTP code"
// @Success      200   {object}  shared.APIResponse{data=auth.AuthResponse}
// @Failure      400   {object}  shared.ErrorResponse
// @Failure      401   {object}  shared.ErrorResponse
// @Header       200   {string}  Set-Cookie  "refresh_token=<token>; Path=/api/v1/auth; HttpOnly; Secure; SameSite=Strict"
// @Router       /auth/login/otp [post]
func (h *AuthHandler) LoginOTP(w http.ResponseWriter, r *http.Request) {
	var req auth.LoginOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.BadRequest(w, r, "auth", "invalid request body")
		return
	}

	if errs := validateLoginOTP(&req); len(errs) > 0 {
		shared.ValidationError(w, r, "auth", "validation failed", errs)
		return
	}

	lc := loginContextFromRequest(r)
	resp, refreshToken, err := h.Service.LoginWithOTP(r.Context(), &req, lc)
	if err != nil {
		shared.Unauthorized(w, r, "auth", err.Error())
		return
	}

	h.setRefreshTokenCookie(w, refreshToken)
	shared.OK(w, r, resp)
}

// RequestOTP godoc
// @Summary      Request OTP code via WhatsApp
// @Description  Send a 6-digit OTP to the user's WhatsApp number via Zenziva. Valid for 5 minutes, max 3 attempts.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      auth.RequestOTPRequest  true  "WhatsApp number and purpose"
// @Success      200   {object}  shared.APIResponse{data=auth.MessageResponse}
// @Failure      400   {object}  shared.ErrorResponse
// @Router       /auth/otp/request [post]
func (h *AuthHandler) RequestOTP(w http.ResponseWriter, r *http.Request) {
	var req auth.RequestOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.BadRequest(w, r, "auth", "invalid request body")
		return
	}

	if req.WhatsApp == "" || req.Purpose == "" {
		shared.BadRequest(w, r, "auth", "whatsapp and purpose are required")
		return
	}

	if err := h.Service.RequestOTP(r.Context(), &req); err != nil {
		shared.BadRequest(w, r, "auth", err.Error())
		return
	}

	shared.OK(w, r, auth.MessageResponse{Message: "OTP sent successfully"})
}

// RefreshToken godoc
// @Summary      Refresh access token
// @Description  Exchange refresh token (from HttpOnly cookie) for a new access token + rotated refresh token.
// @Tags         Auth
// @Produce      json
// @Success      200   {object}  shared.APIResponse{data=auth.AuthResponse}
// @Failure      401   {object}  shared.ErrorResponse
// @Header       200   {string}  Set-Cookie  "refresh_token=<new_token>; Path=/api/v1/auth; HttpOnly; Secure; SameSite=Strict"
// @Router       /auth/refresh [post]
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	refreshToken := middleware.GetRefreshToken(r)
	if refreshToken == "" {
		shared.Unauthorized(w, r, "auth", "missing refresh token")
		return
	}

	lc := loginContextFromRequest(r)
	resp, newRefreshToken, err := h.Service.RefreshToken(r.Context(), refreshToken, lc)
	if err != nil {
		h.clearRefreshTokenCookie(w)
		shared.Unauthorized(w, r, "auth", err.Error())
		return
	}

	h.setRefreshTokenCookie(w, newRefreshToken)
	shared.OK(w, r, resp)
}

// Logout godoc
// @Summary      Logout current session
// @Description  Blacklist the access token and delete the current session. Clears refresh token cookie.
// @Tags         Auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  shared.APIResponse{data=auth.MessageResponse}
// @Failure      400  {object}  shared.ErrorResponse
// @Failure      401  {object}  shared.ErrorResponse
// @Router       /auth/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	accessToken := middleware.GetAccessToken(r)
	refreshToken := middleware.GetRefreshToken(r)

	if err := h.Service.Logout(r.Context(), accessToken, refreshToken); err != nil {
		shared.BadRequest(w, r, "auth", err.Error())
		return
	}

	h.clearRefreshTokenCookie(w)
	shared.OK(w, r, auth.MessageResponse{Message: "Logged out successfully"})
}

// LogoutAll godoc
// @Summary      Logout all sessions
// @Description  Blacklist the access token and delete ALL sessions for the user. Clears refresh token cookie.
// @Tags         Auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  shared.APIResponse{data=auth.MessageResponse}
// @Failure      400  {object}  shared.ErrorResponse
// @Failure      401  {object}  shared.ErrorResponse
// @Router       /auth/logout-all [post]
func (h *AuthHandler) LogoutAll(w http.ResponseWriter, r *http.Request) {
	accessToken := middleware.GetAccessToken(r)

	if err := h.Service.LogoutAll(r.Context(), accessToken); err != nil {
		shared.BadRequest(w, r, "auth", err.Error())
		return
	}

	h.clearRefreshTokenCookie(w)
	shared.OK(w, r, auth.MessageResponse{Message: "All sessions revoked"})
}

// Me godoc
// @Summary      Get current user
// @Description  Returns the authenticated user's profile information.
// @Tags         Auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  shared.APIResponse{data=auth.UserResponse}
// @Failure      401  {object}  shared.ErrorResponse
// @Failure      404  {object}  shared.ErrorResponse
// @Router       /auth/me [get]
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		shared.Unauthorized(w, r, "auth", "unauthorized")
		return
	}

	user, err := h.Service.GetMe(r.Context(), userID)
	if err != nil {
		shared.NotFound(w, r, "auth", "user not found")
		return
	}

	shared.OK(w, r, auth.UserToResponse(user))
}

// GetSessions godoc
// @Summary      List active sessions
// @Description  Returns all active sessions for the authenticated user.
// @Tags         Auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  shared.APIResponse{data=[]auth.SessionResponse}
// @Failure      401  {object}  shared.ErrorResponse
// @Failure      500  {object}  shared.ErrorResponse
// @Router       /auth/sessions [get]
func (h *AuthHandler) GetSessions(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	sessions, err := h.Service.GetSessions(r.Context(), userID)
	if err != nil {
		shared.InternalError(w, r, "auth", "failed to get sessions")
		return
	}

	var resp []auth.SessionResponse
	for i := range sessions {
		resp = append(resp, auth.SessionToResponse(&sessions[i]))
	}
	if resp == nil {
		resp = []auth.SessionResponse{}
	}

	shared.OK(w, r, resp)
}

// RevokeSession godoc
// @Summary      Revoke a specific session
// @Description  Delete a specific session by ID. The user can only revoke their own sessions.
// @Tags         Auth
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Session ID (UUID)"
// @Success      200  {object}  shared.APIResponse{data=auth.MessageResponse}
// @Failure      400  {object}  shared.ErrorResponse
// @Failure      401  {object}  shared.ErrorResponse
// @Router       /auth/sessions/{id} [delete]
func (h *AuthHandler) RevokeSession(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	sessionID := chi.URLParam(r, "id")

	if err := h.Service.RevokeSession(r.Context(), userID, sessionID); err != nil {
		shared.BadRequest(w, r, "auth", err.Error())
		return
	}

	shared.OK(w, r, auth.MessageResponse{Message: "Session revoked"})
}

// VerifyOTP godoc
// @Summary      Verify OTP code
// @Description  Verify an OTP code to activate the user account (email or whatsapp verification).
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      auth.VerifyOTPRequest  true  "Verification data"
// @Success      200   {object}  shared.APIResponse{data=auth.MessageResponse}
// @Failure      400   {object}  shared.ErrorResponse
// @Router       /auth/verify [post]
func (h *AuthHandler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req auth.VerifyOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.BadRequest(w, r, "auth", "invalid request body")
		return
	}

	if req.Identifier == "" || req.OTP == "" || req.Purpose == "" {
		shared.BadRequest(w, r, "auth", "identifier, otp, and purpose are required")
		return
	}

	if err := h.Service.VerifyOTP(r.Context(), &req); err != nil {
		shared.BadRequest(w, r, "auth", err.Error())
		return
	}

	shared.OK(w, r, auth.MessageResponse{Message: "Verification successful"})
}

// ChangePassword godoc
// @Summary      Change password
// @Description  Change the authenticated user's password. Requires old password for verification.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      auth.ChangePasswordRequest  true  "Password change data"
// @Success      200   {object}  shared.APIResponse{data=auth.MessageResponse}
// @Failure      400   {object}  shared.ErrorResponse
// @Failure      401   {object}  shared.ErrorResponse
// @Router       /auth/password/change [post]
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var req auth.ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.BadRequest(w, r, "auth", "invalid request body")
		return
	}

	if req.OldPassword == "" || req.NewPassword == "" {
		shared.BadRequest(w, r, "auth", "old_password and new_password are required")
		return
	}

	if err := h.Service.ChangePassword(r.Context(), userID, &req); err != nil {
		shared.BadRequest(w, r, "auth", err.Error())
		return
	}

	shared.OK(w, r, auth.MessageResponse{Message: "Password changed successfully"})
}

// ResetPassword godoc
// @Summary      Reset password with OTP
// @Description  Reset password using an OTP code sent to email or WhatsApp. No authentication required.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      auth.ResetPasswordRequest  true  "Reset password data"
// @Success      200   {object}  shared.APIResponse{data=auth.MessageResponse}
// @Failure      400   {object}  shared.ErrorResponse
// @Router       /auth/password/reset [post]
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req auth.ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.BadRequest(w, r, "auth", "invalid request body")
		return
	}

	if req.Identifier == "" || req.OTP == "" || req.NewPassword == "" {
		shared.BadRequest(w, r, "auth", "identifier, otp, and new_password are required")
		return
	}

	if err := h.Service.ResetPassword(r.Context(), &req); err != nil {
		shared.BadRequest(w, r, "auth", err.Error())
		return
	}

	shared.OK(w, r, auth.MessageResponse{Message: "Password reset successful"})
}

// --- Cookie helpers ---

func (h *AuthHandler) setRefreshTokenCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    token,
		Path:     "/api/v1/auth",
		MaxAge:   int(h.RefreshTokenExpiry.Seconds()),
		Expires:  time.Now().Add(h.RefreshTokenExpiry),
		HttpOnly: true,
		Secure:   h.SecureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *AuthHandler) clearRefreshTokenCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/api/v1/auth",
		MaxAge:   -1,
		Expires:  time.Now().Add(-1 * time.Hour),
		HttpOnly: true,
		Secure:   h.SecureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}

// --- Validation helpers ---

func validateRegister(req *auth.RegisterRequest) []shared.FieldError {
	var errs []shared.FieldError

	if req.Email == nil && req.WhatsApp == nil {
		errs = append(errs, shared.FieldError{Field: "email", Message: "email or whatsapp is required"})
	}
	if req.Email != nil && *req.Email != "" {
		if msg := shared.ValidateEmail(*req.Email); msg != "" {
			errs = append(errs, shared.FieldError{Field: "email", Message: msg})
		}
	}
	if req.WhatsApp != nil && *req.WhatsApp != "" {
		if msg := shared.ValidatePhone(*req.WhatsApp); msg != "" {
			errs = append(errs, shared.FieldError{Field: "whatsapp", Message: msg})
		}
	}
	if msg := shared.ValidateMinLength(req.Password, 8, "password"); msg != "" {
		errs = append(errs, shared.FieldError{Field: "password", Message: msg})
	}

	return errs
}

func validateLoginEmail(req *auth.LoginEmailRequest) []shared.FieldError {
	var errs []shared.FieldError

	if req.Email == "" {
		errs = append(errs, shared.FieldError{Field: "email", Message: "email is required"})
	} else if msg := shared.ValidateEmail(req.Email); msg != "" {
		errs = append(errs, shared.FieldError{Field: "email", Message: msg})
	}
	if req.Password == "" {
		errs = append(errs, shared.FieldError{Field: "password", Message: "password is required"})
	}

	return errs
}

func validateLoginOTP(req *auth.LoginOTPRequest) []shared.FieldError {
	var errs []shared.FieldError

	if req.WhatsApp == "" {
		errs = append(errs, shared.FieldError{Field: "whatsapp", Message: "whatsapp is required"})
	}
	if req.OTP == "" {
		errs = append(errs, shared.FieldError{Field: "otp", Message: "otp is required"})
	}

	return errs
}

func loginContextFromRequest(r *http.Request) *auth.LoginContext {
	return &auth.LoginContext{
		IPAddress:  middleware.GetClientIP(r),
		UserAgent:  r.UserAgent(),
		DeviceName: r.Header.Get("X-Device-Name"),
	}
}
