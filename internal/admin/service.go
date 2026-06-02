package admin

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"net/netip"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"codeberg.org/azzet/azzetbe/internal/db"
	rdb "codeberg.org/azzet/azzetbe/internal/redis"
	"codeberg.org/azzet/azzetbe/internal/shared"
)

var ErrAdminNotFound = errors.New("admin not found")

type Service struct {
	Queries *db.Queries
	Redis   *rdb.Redis
	JWT     *shared.JWTService
}

func NewService(queries *db.Queries, redis *rdb.Redis, jwt *shared.JWTService) *Service {
	return &Service{
		Queries: queries,
		Redis:   redis,
		JWT:     jwt,
	}
}

// Login validates email + password, returns MFA requirement
func (s *Service) Login(ctx context.Context, req *LoginRequest, ip string) (*LoginResponse, error) {
	admin, err := s.Queries.GetAdminByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("invalid credentials")
		}
		return nil, fmt.Errorf("invalid credentials")
	}

	if admin.Status != AdminStatusActive {
		return nil, fmt.Errorf("account is suspended")
	}

	if !shared.VerifyPassword(admin.PasswordHash, req.Password) {
		return nil, fmt.Errorf("invalid credentials")
	}

	if admin.MfaEnabled {
		// Generate temp MFA token (stored in Redis, 5 min TTL)
		mfaToken := uuid.New().String()
		err := s.Redis.Set(ctx, "mfa:"+mfaToken, admin.ID.String(), 5*time.Minute).Err()
		if err != nil {
			return nil, fmt.Errorf("failed to create MFA session")
		}

		return &LoginResponse{
			RequiresMFA: true,
			MFAToken:    &mfaToken,
		}, nil
	}

	// MFA not enabled — issue setup-scoped token (not full admin access)
	sessionID := uuid.New().String()
	accessToken, err := s.JWT.GenerateAccessTokenWithScope(admin.ID.String(), sessionID, shared.TokenScopeMFASetup)
	if err != nil {
		return nil, err
	}
	expiresIn := int(AdminAccessTokenExpiry.Seconds())
	resp := AdminToResponse(&admin)
	return &LoginResponse{
		RequiresMFA: false,
		AccessToken: &accessToken,
		ExpiresIn:   &expiresIn,
		Admin:       &resp,
	}, nil
}

// VerifyMFA validates TOTP code and issues full access token
func (s *Service) VerifyMFA(ctx context.Context, req *MFAVerifyRequest, ip string) (*AuthResponse, string, error) {
	// Get admin ID from temp MFA token
	adminIDStr, err := s.Redis.Get(ctx, "mfa:"+req.MFAToken).Result()
	if err != nil {
		return nil, "", fmt.Errorf("invalid or expired MFA token")
	}

	adminID, err := uuid.Parse(adminIDStr)
	if err != nil {
		return nil, "", fmt.Errorf("invalid MFA session")
	}

	admin, err := s.Queries.GetAdminByID(ctx, adminID)
	if err != nil {
		return nil, "", fmt.Errorf("admin not found")
	}

	// Rate limit MFA attempts per token
	failKey := "admin:mfa:fail:" + req.MFAToken
	if n, _ := s.Redis.Get(ctx, failKey).Int64(); n >= 5 {
		return nil, "", fmt.Errorf("too many MFA attempts, try again later")
	}

	// Validate TOTP code
	if !admin.MfaSecret.Valid || !ValidateMFACode(admin.MfaSecret.String, req.Code) {
		_ = s.Redis.Incr(ctx, failKey).Err()
		_ = s.Redis.Expire(ctx, failKey, 5*time.Minute).Err()
		return nil, "", fmt.Errorf("invalid MFA code")
	}
	_ = s.Redis.Del(ctx, failKey)

	// Delete temp MFA token
	_ = s.Redis.Del(ctx, "mfa:"+req.MFAToken)

	// Update last login
	if parsedIP, err := netip.ParseAddr(ip); err == nil {
		_ = s.Queries.UpdateAdminLastLogin(ctx, db.UpdateAdminLastLoginParams{
			ID:          adminID,
			LastLoginIp: &parsedIP,
		})
	}

	// Issue tokens
	return s.issueTokens(ctx, &admin)
}

// SetupMFA generates a new TOTP secret for the admin
func (s *Service) SetupMFA(ctx context.Context, adminID string) (*MFASetupResponse, error) {
	id, err := uuid.Parse(adminID)
	if err != nil {
		return nil, ErrAdminNotFound
	}

	admin, err := s.Queries.GetAdminByID(ctx, id)
	if err != nil {
		return nil, ErrAdminNotFound
	}

	if admin.MfaEnabled {
		return nil, fmt.Errorf("MFA is already enabled")
	}

	secret, qrURL, err := GenerateMFASecret(admin.Email)
	if err != nil {
		return nil, err
	}

	// Store secret (not yet enabled until confirmed)
	_ = s.Queries.UpdateAdminMFA(ctx, db.UpdateAdminMFAParams{
		ID:         id,
		MfaSecret:  pgtype.Text{String: secret, Valid: true},
		MfaEnabled: false,
	})

	return &MFASetupResponse{
		Secret: secret,
		QRCode: qrURL,
	}, nil
}

// ConfirmMFASetup verifies the first TOTP code and enables MFA
func (s *Service) ConfirmMFASetup(ctx context.Context, adminID string, req *MFASetupConfirmRequest) (*AuthResponse, string, error) {
	id, err := uuid.Parse(adminID)
	if err != nil {
		return nil, "", ErrAdminNotFound
	}

	admin, err := s.Queries.GetAdminByID(ctx, id)
	if err != nil {
		return nil, "", ErrAdminNotFound
	}

	if !admin.MfaSecret.Valid {
		return nil, "", fmt.Errorf("MFA setup not initiated")
	}

	if !ValidateMFACode(admin.MfaSecret.String, req.Code) {
		return nil, "", fmt.Errorf("invalid MFA code")
	}

	// Enable MFA
	_ = s.Queries.UpdateAdminMFA(ctx, db.UpdateAdminMFAParams{
		ID:         id,
		MfaSecret:  admin.MfaSecret,
		MfaEnabled: true,
	})

	// Issue full tokens
	admin.MfaEnabled = true
	return s.issueTokens(ctx, &admin)
}

// Logout blacklists the admin access token and deletes the Redis session
func (s *Service) Logout(ctx context.Context, accessToken string) error {
	claims, err := s.JWT.ValidateAccessToken(accessToken)
	if err != nil {
		return fmt.Errorf("invalid access token")
	}
	_ = s.Redis.Set(ctx, "admin:blacklist:"+claims.JTI, claims.UserID, AdminAccessTokenExpiry).Err()
	if claims.SessionID != "" {
		_ = s.Redis.Del(ctx, adminSessionKey(claims.SessionID))
	}
	return nil
}

// GetMe returns the current admin profile
func (s *Service) GetMe(ctx context.Context, adminID string) (*db.PlatformAdmin, error) {
	id, err := uuid.Parse(adminID)
	if err != nil {
		return nil, ErrAdminNotFound
	}
	admin, err := s.Queries.GetAdminByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAdminNotFound
		}
		return nil, err
	}
	return &admin, nil
}

// InviteAdmin creates a new admin (SUPER_ADMIN only)
func (s *Service) InviteAdmin(ctx context.Context, req *InviteAdminRequest) (*db.PlatformAdmin, error) {
	if !IsValidRole(req.Role) {
		return nil, fmt.Errorf("invalid role")
	}
	if len(req.Password) < 12 {
		return nil, fmt.Errorf("password must be at least 12 characters")
	}

	exists, err := s.Queries.ExistsAdminByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("email already registered")
	}

	hash, err := shared.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	admin, err := s.Queries.CreateAdmin(ctx, db.CreateAdminParams{
		ID:           uuid.New(),
		Email:        req.Email,
		PasswordHash: hash,
		Name:         req.Name,
		Role:         req.Role,
		Status:       AdminStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create admin: %w", err)
	}

	return &admin, nil
}

// ListAdmins returns all admins
func (s *Service) ListAdmins(ctx context.Context) ([]db.PlatformAdmin, error) {
	return s.Queries.ListAdmins(ctx)
}

// UpdateAdmin updates an admin's role/status
func (s *Service) UpdateAdmin(ctx context.Context, adminID string, req *UpdateAdminRequest) error {
	id, err := uuid.Parse(adminID)
	if err != nil {
		return ErrAdminNotFound
	}

	admin, err := s.Queries.GetAdminByID(ctx, id)
	if err != nil {
		return ErrAdminNotFound
	}

	name := admin.Name
	role := admin.Role
	status := admin.Status

	if req.Name != nil {
		name = *req.Name
	}
	if req.Role != nil {
		if !IsValidRole(*req.Role) {
			return fmt.Errorf("invalid role")
		}
		role = *req.Role
	}
	if req.Status != nil {
		if *req.Status != AdminStatusActive && *req.Status != AdminStatusSuspended {
			return fmt.Errorf("invalid status")
		}
		status = *req.Status
	}

	if err := s.guardLastSuperAdmin(ctx, admin, role, status); err != nil {
		return err
	}

	return s.Queries.UpdateAdmin(ctx, db.UpdateAdminParams{
		ID:     id,
		Name:   name,
		Role:   role,
		Status: status,
	})
}

// DeleteAdmin soft-deletes an admin
func (s *Service) DeleteAdmin(ctx context.Context, adminID string) error {
	id, err := uuid.Parse(adminID)
	if err != nil {
		return ErrAdminNotFound
	}
	admin, err := s.Queries.GetAdminByID(ctx, id)
	if err != nil {
		return ErrAdminNotFound
	}
	if err := s.guardLastSuperAdmin(ctx, admin, admin.Role, AdminStatusSuspended); err != nil {
		return err
	}
	return s.Queries.DeleteAdmin(ctx, id)
}

func (s *Service) guardLastSuperAdmin(ctx context.Context, admin db.PlatformAdmin, newRole, newStatus string) error {
	if admin.Role != RoleSuperAdmin {
		return nil
	}
	if newRole == RoleSuperAdmin && newStatus == AdminStatusActive {
		return nil
	}

	admins, err := s.Queries.ListAdmins(ctx)
	if err != nil {
		return fmt.Errorf("failed to verify super admin count: %w", err)
	}
	activeSuper := 0
	for _, a := range admins {
		if a.Role == RoleSuperAdmin && a.Status == AdminStatusActive {
			activeSuper++
		}
	}
	if activeSuper <= 1 {
		return fmt.Errorf("cannot remove or suspend the last active SUPER_ADMIN")
	}
	return nil
}

// IsTokenBlacklisted checks if an admin token is blacklisted
func (s *Service) IsTokenBlacklisted(ctx context.Context, jti string) (bool, error) {
	return s.Redis.Exists(ctx, "admin:blacklist:"+jti)
}

// RefreshToken issues new tokens from refresh token cookie
func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*AuthResponse, string, error) {
	claims, err := s.JWT.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, "", fmt.Errorf("invalid refresh token")
	}

	adminID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, "", fmt.Errorf("invalid token")
	}

	admin, err := s.Queries.GetAdminByID(ctx, adminID)
	if err != nil {
		return nil, "", fmt.Errorf("admin not found")
	}

	if admin.Status != AdminStatusActive {
		return nil, "", fmt.Errorf("account is suspended")
	}

	if claims.Scope == shared.TokenScopeMFASetup {
		return nil, "", fmt.Errorf("invalid refresh token")
	}

	tokenHash := shared.HashOTP(refreshToken)
	storedHash, err := s.Redis.Get(ctx, adminSessionKey(claims.SessionID)).Result()
	if err != nil || subtle.ConstantTimeCompare([]byte(storedHash), []byte(tokenHash)) != 1 {
		return nil, "", fmt.Errorf("invalid refresh token")
	}

	_ = s.Redis.Del(ctx, adminSessionKey(claims.SessionID))

	return s.issueTokens(ctx, &admin)
}

// --- Private helpers ---

func adminSessionKey(sessionID string) string {
	return "admin:session:" + sessionID
}

func (s *Service) issueTokens(ctx context.Context, admin *db.PlatformAdmin) (*AuthResponse, string, error) {
	sessionID := uuid.New().String()

	accessToken, err := s.JWT.GenerateAccessToken(admin.ID.String(), sessionID)
	if err != nil {
		return nil, "", err
	}

	refreshToken, err := s.JWT.GenerateRefreshToken(admin.ID.String(), sessionID)
	if err != nil {
		return nil, "", err
	}

	tokenHash := shared.HashOTP(refreshToken)
	_ = s.Redis.Set(ctx, adminSessionKey(sessionID), tokenHash, AdminRefreshTokenExpiry).Err()

	resp := &AuthResponse{
		AccessToken: accessToken,
		ExpiresIn:   int(AdminAccessTokenExpiry.Seconds()),
		Admin:       AdminToResponse(admin),
	}

	return resp, refreshToken, nil
}

// AdminToResponse converts db model to response DTO
func AdminToResponse(a *db.PlatformAdmin) AdminResponse {
	var lastLogin *string
	if a.LastLoginAt != nil {
		t := a.LastLoginAt.Format(time.RFC3339)
		lastLogin = &t
	}

	return AdminResponse{
		ID:         a.ID.String(),
		Email:      a.Email,
		Name:       a.Name,
		Role:       a.Role,
		MFAEnabled: a.MfaEnabled,
		Status:     a.Status,
		LastLogin:  lastLogin,
		CreatedAt:  a.CreatedAt.Format(time.RFC3339),
	}
}
