package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"codeberg.org/azzet/azzetbe/internal/admin"
	"codeberg.org/azzet/azzetbe/internal/api/middleware"
	"codeberg.org/azzet/azzetbe/internal/shared"
)

type AdminHandler struct {
	Service            *admin.Service
	RefreshTokenExpiry time.Duration
	SecureCookie       bool
}

func NewAdminHandler(service *admin.Service, refreshTokenExpiry time.Duration, secureCookie bool) *AdminHandler {
	return &AdminHandler{
		Service:            service,
		RefreshTokenExpiry: refreshTokenExpiry,
		SecureCookie:       secureCookie,
	}
}

// Login godoc
// @Summary      Admin login (step 1)
// @Description  Authenticate admin with email + password. If MFA enabled, returns mfa_token for step 2.
// @Tags         Admin Auth
// @Accept       json
// @Produce      json
// @Param        body  body      admin.LoginRequest  true  "Admin credentials"
// @Success      200   {object}  shared.APIResponse{data=admin.LoginResponse}
// @Failure      401   {object}  shared.ErrorResponse
// @Router       /admin/auth/login [post]
func (h *AdminHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req admin.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.BadRequest(w, r, "admin", "invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		shared.BadRequest(w, r, "admin", "email and password are required")
		return
	}

	ip := middleware.GetClientIP(r)
	resp, err := h.Service.Login(r.Context(), &req, ip)
	if err != nil {
		shared.Unauthorized(w, r, "admin", err.Error())
		return
	}

	shared.OK(w, r, resp)
}

// VerifyMFA godoc
// @Summary      Verify MFA code (step 2)
// @Description  Verify TOTP code after password authentication. Returns access token and sets refresh cookie.
// @Tags         Admin Auth
// @Accept       json
// @Produce      json
// @Param        body  body      admin.MFAVerifyRequest  true  "MFA token + TOTP code"
// @Success      200   {object}  shared.APIResponse{data=admin.AuthResponse}
// @Failure      401   {object}  shared.ErrorResponse
// @Header       200   {string}  Set-Cookie  "admin_refresh_token=<token>; Path=/admin; HttpOnly; Secure; SameSite=Strict"
// @Router       /admin/auth/mfa/verify [post]
func (h *AdminHandler) VerifyMFA(w http.ResponseWriter, r *http.Request) {
	var req admin.MFAVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.BadRequest(w, r, "admin", "invalid request body")
		return
	}

	if req.MFAToken == "" || req.Code == "" {
		shared.BadRequest(w, r, "admin", "mfa_token and code are required")
		return
	}

	ip := middleware.GetClientIP(r)
	resp, refreshToken, err := h.Service.VerifyMFA(r.Context(), &req, ip)
	if err != nil {
		shared.Unauthorized(w, r, "admin", err.Error())
		return
	}

	h.setRefreshTokenCookie(w, refreshToken)
	shared.OK(w, r, resp)
}

// SetupMFA godoc
// @Summary      Setup MFA (first time)
// @Description  Generate TOTP secret and QR code URL. Admin must confirm with first code.
// @Tags         Admin Auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  shared.APIResponse{data=admin.MFASetupResponse}
// @Failure      400  {object}  shared.ErrorResponse
// @Failure      401  {object}  shared.ErrorResponse
// @Router       /admin/auth/mfa/setup [post]
func (h *AdminHandler) SetupMFA(w http.ResponseWriter, r *http.Request) {
	adminID := middleware.GetAdminID(r.Context())
	if adminID == "" {
		shared.Unauthorized(w, r, "admin", "unauthorized")
		return
	}

	resp, err := h.Service.SetupMFA(r.Context(), adminID)
	if err != nil {
		shared.BadRequest(w, r, "admin", err.Error())
		return
	}

	shared.OK(w, r, resp)
}

// ConfirmMFASetup godoc
// @Summary      Confirm MFA setup
// @Description  Confirm MFA setup by providing the first TOTP code from authenticator app. Enables MFA and issues tokens.
// @Tags         Admin Auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      admin.MFASetupConfirmRequest  true  "First TOTP code"
// @Success      200   {object}  shared.APIResponse{data=admin.AuthResponse}
// @Failure      400   {object}  shared.ErrorResponse
// @Failure      401   {object}  shared.ErrorResponse
// @Header       200   {string}  Set-Cookie  "admin_refresh_token=<token>; Path=/admin; HttpOnly; Secure; SameSite=Strict"
// @Router       /admin/auth/mfa/confirm [post]
func (h *AdminHandler) ConfirmMFASetup(w http.ResponseWriter, r *http.Request) {
	adminID := middleware.GetAdminID(r.Context())
	if adminID == "" {
		shared.Unauthorized(w, r, "admin", "unauthorized")
		return
	}

	var req admin.MFASetupConfirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.BadRequest(w, r, "admin", "invalid request body")
		return
	}

	if req.Code == "" {
		shared.BadRequest(w, r, "admin", "code is required")
		return
	}

	resp, refreshToken, err := h.Service.ConfirmMFASetup(r.Context(), adminID, &req)
	if err != nil {
		shared.BadRequest(w, r, "admin", err.Error())
		return
	}

	h.setRefreshTokenCookie(w, refreshToken)
	shared.OK(w, r, resp)
}

// RefreshToken godoc
// @Summary      Refresh admin access token
// @Description  Exchange refresh token (from HttpOnly cookie) for new tokens.
// @Tags         Admin Auth
// @Produce      json
// @Success      200   {object}  shared.APIResponse{data=admin.AuthResponse}
// @Failure      401   {object}  shared.ErrorResponse
// @Header       200   {string}  Set-Cookie  "admin_refresh_token=<new_token>; Path=/admin; HttpOnly; Secure; SameSite=Strict"
// @Router       /admin/auth/refresh [post]
func (h *AdminHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("admin_refresh_token")
	if err != nil || cookie.Value == "" {
		shared.Unauthorized(w, r, "admin", "missing refresh token")
		return
	}

	resp, newRefreshToken, err := h.Service.RefreshToken(r.Context(), cookie.Value)
	if err != nil {
		h.clearRefreshTokenCookie(w)
		shared.Unauthorized(w, r, "admin", err.Error())
		return
	}

	h.setRefreshTokenCookie(w, newRefreshToken)
	shared.OK(w, r, resp)
}

// Logout godoc
// @Summary      Admin logout
// @Description  Blacklist the admin access token and clear refresh cookie.
// @Tags         Admin Auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  shared.APIResponse{data=admin.MessageResponse}
// @Failure      401  {object}  shared.ErrorResponse
// @Router       /admin/auth/logout [post]
func (h *AdminHandler) Logout(w http.ResponseWriter, r *http.Request) {
	accessToken := middleware.GetAccessToken(r)

	if err := h.Service.Logout(r.Context(), accessToken); err != nil {
		shared.BadRequest(w, r, "admin", err.Error())
		return
	}

	h.clearRefreshTokenCookie(w)
	shared.OK(w, r, admin.MessageResponse{Message: "Logged out successfully"})
}

// Me godoc
// @Summary      Get current admin profile
// @Description  Returns the authenticated admin's profile.
// @Tags         Admin Auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  shared.APIResponse{data=admin.AdminResponse}
// @Failure      401  {object}  shared.ErrorResponse
// @Router       /admin/auth/me [get]
func (h *AdminHandler) Me(w http.ResponseWriter, r *http.Request) {
	adminID := middleware.GetAdminID(r.Context())
	if adminID == "" {
		shared.Unauthorized(w, r, "admin", "unauthorized")
		return
	}

	a, err := h.Service.GetMe(r.Context(), adminID)
	if err != nil {
		shared.NotFound(w, r, "admin", "admin not found")
		return
	}

	shared.OK(w, r, admin.AdminToResponse(a))
}

// InviteAdmin godoc
// @Summary      Invite new admin
// @Description  Create a new platform admin account. SUPER_ADMIN only.
// @Tags         Admin Management
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      admin.InviteAdminRequest  true  "New admin data"
// @Success      201   {object}  shared.APIResponse{data=admin.AdminResponse}
// @Failure      400   {object}  shared.ErrorResponse
// @Failure      401   {object}  shared.ErrorResponse
// @Failure      403   {object}  shared.ErrorResponse
// @Router       /admin/admins [post]
func (h *AdminHandler) InviteAdmin(w http.ResponseWriter, r *http.Request) {
	var req admin.InviteAdminRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.BadRequest(w, r, "admin", "invalid request body")
		return
	}

	if req.Email == "" || req.Name == "" || req.Role == "" || req.Password == "" {
		shared.BadRequest(w, r, "admin", "email, name, role, and password are required")
		return
	}

	a, err := h.Service.InviteAdmin(r.Context(), &req)
	if err != nil {
		shared.BadRequest(w, r, "admin", err.Error())
		return
	}

	shared.Created(w, r, admin.AdminToResponse(a))
}

// ListAdmins godoc
// @Summary      List all admins
// @Description  Returns all platform admins. SUPER_ADMIN only.
// @Tags         Admin Management
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  shared.APIResponse{data=[]admin.AdminResponse}
// @Failure      401  {object}  shared.ErrorResponse
// @Failure      403  {object}  shared.ErrorResponse
// @Router       /admin/admins [get]
func (h *AdminHandler) ListAdmins(w http.ResponseWriter, r *http.Request) {
	admins, err := h.Service.ListAdmins(r.Context())
	if err != nil {
		shared.InternalError(w, r, "admin", "failed to list admins")
		return
	}

	var resp []admin.AdminResponse
	for i := range admins {
		resp = append(resp, admin.AdminToResponse(&admins[i]))
	}
	if resp == nil {
		resp = []admin.AdminResponse{}
	}

	shared.OK(w, r, resp)
}

// UpdateAdmin godoc
// @Summary      Update admin
// @Description  Update admin role or status. SUPER_ADMIN only.
// @Tags         Admin Management
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                  true  "Admin ID (UUID)"
// @Param        body  body      admin.UpdateAdminRequest  true  "Update data"
// @Success      200   {object}  shared.APIResponse{data=admin.MessageResponse}
// @Failure      400   {object}  shared.ErrorResponse
// @Failure      401   {object}  shared.ErrorResponse
// @Failure      403   {object}  shared.ErrorResponse
// @Router       /admin/admins/{id} [patch]
func (h *AdminHandler) UpdateAdmin(w http.ResponseWriter, r *http.Request) {
	adminID := chi.URLParam(r, "id")

	var req admin.UpdateAdminRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.BadRequest(w, r, "admin", "invalid request body")
		return
	}

	if err := h.Service.UpdateAdmin(r.Context(), adminID, &req); err != nil {
		shared.BadRequest(w, r, "admin", err.Error())
		return
	}

	shared.OK(w, r, admin.MessageResponse{Message: "Admin updated"})
}

// DeleteAdmin godoc
// @Summary      Deactivate admin
// @Description  Soft-delete an admin account. SUPER_ADMIN only.
// @Tags         Admin Management
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Admin ID (UUID)"
// @Success      200  {object}  shared.APIResponse{data=admin.MessageResponse}
// @Failure      400  {object}  shared.ErrorResponse
// @Failure      401  {object}  shared.ErrorResponse
// @Failure      403  {object}  shared.ErrorResponse
// @Router       /admin/admins/{id} [delete]
func (h *AdminHandler) DeleteAdmin(w http.ResponseWriter, r *http.Request) {
	adminID := chi.URLParam(r, "id")

	if err := h.Service.DeleteAdmin(r.Context(), adminID); err != nil {
		shared.BadRequest(w, r, "admin", err.Error())
		return
	}

	shared.OK(w, r, admin.MessageResponse{Message: "Admin deactivated"})
}

// --- Cookie helpers ---

func (h *AdminHandler) setRefreshTokenCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "admin_refresh_token",
		Value:    token,
		Path:     "/api/v1/admin",
		MaxAge:   int(h.RefreshTokenExpiry.Seconds()),
		Expires:  time.Now().Add(h.RefreshTokenExpiry),
		HttpOnly: true,
		Secure:   h.SecureCookie,
		SameSite: http.SameSiteStrictMode,
	})
}

func (h *AdminHandler) clearRefreshTokenCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "admin_refresh_token",
		Value:    "",
		Path:     "/api/v1/admin",
		MaxAge:   -1,
		Expires:  time.Now().Add(-1 * time.Hour),
		HttpOnly: true,
		Secure:   h.SecureCookie,
		SameSite: http.SameSiteStrictMode,
	})
}
