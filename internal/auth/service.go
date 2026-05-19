package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"codeberg.org/azzet/azzetbe/internal/shared"
)

type Service struct {
	DB      *pgxpool.Pool
	JWT     *shared.JWTService
	OTP     *shared.OTPService
	Zenziva *shared.ZenzivaClient
	Email   *shared.EmailOTPSender
	Config  *ServiceConfig
}

type ServiceConfig struct {
	AccessTokenExpiry  time.Duration
	RefreshTokenExpiry time.Duration
	OTPExpiry          time.Duration
	OTPMaxAttempts     int
}

func NewService(db *pgxpool.Pool, jwt *shared.JWTService, otp *shared.OTPService, zenziva *shared.ZenzivaClient, email *shared.EmailOTPSender, cfg *ServiceConfig) *Service {
	return &Service{
		DB:      db,
		JWT:     jwt,
		OTP:     otp,
		Zenziva: zenziva,
		Email:   email,
		Config:  cfg,
	}
}

// Register creates a new user with email or whatsapp + password (always required)
func (s *Service) Register(ctx context.Context, req *RegisterRequest) (*User, error) {
	if req.Email == nil && req.WhatsApp == nil {
		return nil, fmt.Errorf("email or whatsapp is required")
	}
	if req.Password == "" {
		return nil, fmt.Errorf("password is required")
	}
	if len(req.Password) < 8 {
		return nil, fmt.Errorf("password must be at least 8 characters")
	}

	// Check duplicates
	if req.Email != nil && *req.Email != "" {
		var exists bool
		err := s.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`, *req.Email).Scan(&exists)
		if err != nil {
			return nil, fmt.Errorf("failed to check email: %w", err)
		}
		if exists {
			return nil, fmt.Errorf("email already registered")
		}
	}
	if req.WhatsApp != nil && *req.WhatsApp != "" {
		var exists bool
		err := s.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE whatsapp = $1)`, *req.WhatsApp).Scan(&exists)
		if err != nil {
			return nil, fmt.Errorf("failed to check whatsapp: %w", err)
		}
		if exists {
			return nil, fmt.Errorf("whatsapp already registered")
		}
	}

	hash, err := shared.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	id := shared.GenerateUUID()
	now := time.Now()

	_, err = s.DB.Exec(ctx, `
		INSERT INTO users (id, email, whatsapp, password_hash, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, id, req.Email, req.WhatsApp, hash, StatusUnverified, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	user := &User{
		ID:        id,
		Email:     req.Email,
		WhatsApp:  req.WhatsApp,
		Status:    StatusUnverified,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Send verification OTP (non-blocking errors)
	if req.WhatsApp != nil && *req.WhatsApp != "" {
		code := s.OTP.Generate()
		if err := s.storeOTP(ctx, *req.WhatsApp, IdentifierTypeWA, OTPPurposeVerifyWA, code); err == nil {
			_ = s.Zenziva.SendOTP(ctx, *req.WhatsApp, code)
		}
	}
	if req.Email != nil && *req.Email != "" {
		code := s.OTP.Generate()
		if err := s.storeOTP(ctx, *req.Email, IdentifierTypeEmail, OTPPurposeVerifyEmail, code); err == nil {
			_ = s.Email.SendOTP(ctx, *req.Email, code)
		}
	}

	return user, nil
}

// LoginWithEmail authenticates user with email + password
func (s *Service) LoginWithEmail(ctx context.Context, req *LoginEmailRequest) (*AuthResponse, string, error) {
	user, err := s.getUserByEmail(ctx, req.Email)
	if err != nil || user == nil {
		return nil, "", fmt.Errorf("invalid credentials")
	}
	if user.PasswordHash == nil || !shared.VerifyPassword(*user.PasswordHash, req.Password) {
		return nil, "", fmt.Errorf("invalid credentials")
	}
	if user.Status == StatusSuspended {
		return nil, "", fmt.Errorf("account is suspended")
	}

	return s.createTokenPair(ctx, user)
}

// LoginWithOTP authenticates user with whatsapp + OTP code
func (s *Service) LoginWithOTP(ctx context.Context, req *LoginOTPRequest) (*AuthResponse, string, error) {
	if err := s.validateOTP(ctx, req.WhatsApp, req.OTP, OTPPurposeLogin); err != nil {
		return nil, "", err
	}

	user, err := s.getUserByWhatsApp(ctx, req.WhatsApp)
	if err != nil || user == nil {
		return nil, "", fmt.Errorf("user not found")
	}
	if user.Status == StatusSuspended {
		return nil, "", fmt.Errorf("account is suspended")
	}

	return s.createTokenPair(ctx, user)
}

// RequestOTP sends an OTP to the user's whatsapp
func (s *Service) RequestOTP(ctx context.Context, req *RequestOTPRequest) error {
	user, err := s.getUserByWhatsApp(ctx, req.WhatsApp)
	if err != nil || user == nil {
		return fmt.Errorf("user not found")
	}

	code := s.OTP.Generate()
	if err := s.storeOTP(ctx, req.WhatsApp, IdentifierTypeWA, req.Purpose, code); err != nil {
		return err
	}

	return s.Zenziva.SendOTP(ctx, req.WhatsApp, code)
}

// RefreshToken rotates the refresh token and issues new access token
func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*AuthResponse, string, error) {
	claims, err := s.JWT.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, "", fmt.Errorf("invalid refresh token")
	}

	// Verify session exists and is valid
	var expiresAt time.Time
	err = s.DB.QueryRow(ctx, `
		SELECT expires_at FROM sessions WHERE id = $1 AND user_id = $2 AND refresh_token = $3
	`, claims.SessionID, claims.UserID, refreshToken).Scan(&expiresAt)
	if err != nil {
		return nil, "", fmt.Errorf("session not found")
	}
	if time.Now().After(expiresAt) {
		return nil, "", fmt.Errorf("session expired")
	}

	// Get user
	user, err := s.getUserByID(ctx, claims.UserID)
	if err != nil || user == nil {
		return nil, "", fmt.Errorf("user not found")
	}

	// Delete old session
	_, _ = s.DB.Exec(ctx, `DELETE FROM sessions WHERE id = $1`, claims.SessionID)

	// Create new token pair
	return s.createTokenPair(ctx, user)
}

// Logout blacklists the access token and deletes the session
func (s *Service) Logout(ctx context.Context, accessToken, refreshToken string) error {
	claims, err := s.JWT.ValidateAccessToken(accessToken)
	if err != nil {
		return fmt.Errorf("invalid access token")
	}

	_ = s.blacklistToken(ctx, claims.JTI, claims.UserID, s.Config.AccessTokenExpiry)

	if refreshToken != "" {
		_, _ = s.DB.Exec(ctx, `DELETE FROM sessions WHERE refresh_token = $1`, refreshToken)
	}

	return nil
}

// LogoutAll blacklists current token and deletes all user sessions
func (s *Service) LogoutAll(ctx context.Context, accessToken string) error {
	claims, err := s.JWT.ValidateAccessToken(accessToken)
	if err != nil {
		return fmt.Errorf("invalid access token")
	}

	_ = s.blacklistToken(ctx, claims.JTI, claims.UserID, s.Config.AccessTokenExpiry)
	_, _ = s.DB.Exec(ctx, `DELETE FROM sessions WHERE user_id = $1`, claims.UserID)

	return nil
}

// GetMe returns the current user
func (s *Service) GetMe(ctx context.Context, userID string) (*User, error) {
	return s.getUserByID(ctx, userID)
}

// GetSessions returns all active sessions for a user
func (s *Service) GetSessions(ctx context.Context, userID string) ([]Session, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT id, user_id, refresh_token, device_name, device_type, ip_address, user_agent, expires_at, last_used_at, created_at
		FROM sessions WHERE user_id = $1 AND expires_at > NOW()
		ORDER BY last_used_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var sess Session
		err := rows.Scan(&sess.ID, &sess.UserID, &sess.RefreshToken, &sess.DeviceName, &sess.DeviceType, &sess.IPAddress, &sess.UserAgent, &sess.ExpiresAt, &sess.LastUsedAt, &sess.CreatedAt)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, sess)
	}
	return sessions, nil
}

// RevokeSession deletes a specific session
func (s *Service) RevokeSession(ctx context.Context, userID, sessionID string) error {
	result, err := s.DB.Exec(ctx, `DELETE FROM sessions WHERE id = $1 AND user_id = $2`, sessionID, userID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("session not found")
	}
	return nil
}

// VerifyOTP verifies an OTP and activates the user
func (s *Service) VerifyOTP(ctx context.Context, req *VerifyOTPRequest) error {
	if err := s.validateOTP(ctx, req.Identifier, req.OTP, req.Purpose); err != nil {
		return err
	}

	switch req.Purpose {
	case OTPPurposeVerifyWA:
		_, err := s.DB.Exec(ctx, `UPDATE users SET whatsapp_verified = TRUE, status = $1, updated_at = NOW() WHERE whatsapp = $2`, StatusActive, req.Identifier)
		return err
	case OTPPurposeVerifyEmail:
		_, err := s.DB.Exec(ctx, `UPDATE users SET email_verified = TRUE, status = $1, updated_at = NOW() WHERE email = $2`, StatusActive, req.Identifier)
		return err
	}
	return nil
}

// ChangePassword changes the user's password
func (s *Service) ChangePassword(ctx context.Context, userID string, req *ChangePasswordRequest) error {
	user, err := s.getUserByID(ctx, userID)
	if err != nil || user == nil {
		return fmt.Errorf("user not found")
	}
	if user.PasswordHash == nil || !shared.VerifyPassword(*user.PasswordHash, req.OldPassword) {
		return fmt.Errorf("invalid old password")
	}
	if len(req.NewPassword) < 8 {
		return fmt.Errorf("new password must be at least 8 characters")
	}

	hash, err := shared.HashPassword(req.NewPassword)
	if err != nil {
		return err
	}

	_, err = s.DB.Exec(ctx, `UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2`, hash, userID)
	return err
}

// ResetPassword resets password using OTP
func (s *Service) ResetPassword(ctx context.Context, req *ResetPasswordRequest) error {
	if err := s.validateOTP(ctx, req.Identifier, req.OTP, OTPPurposeResetPass); err != nil {
		return err
	}
	if len(req.NewPassword) < 8 {
		return fmt.Errorf("new password must be at least 8 characters")
	}

	hash, err := shared.HashPassword(req.NewPassword)
	if err != nil {
		return err
	}

	_, err = s.DB.Exec(ctx, `
		UPDATE users SET password_hash = $1, updated_at = NOW() 
		WHERE email = $2 OR whatsapp = $2
	`, hash, req.Identifier)
	return err
}

// --- Private helpers ---

func (s *Service) createTokenPair(ctx context.Context, user *User) (*AuthResponse, string, error) {
	sessionID := shared.GenerateUUID()

	accessToken, err := s.JWT.GenerateAccessToken(user.ID, sessionID)
	if err != nil {
		return nil, "", err
	}

	refreshToken, err := s.JWT.GenerateRefreshToken(user.ID, sessionID)
	if err != nil {
		return nil, "", err
	}

	_, err = s.DB.Exec(ctx, `
		INSERT INTO sessions (id, user_id, refresh_token, expires_at, last_used_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, sessionID, user.ID, refreshToken, time.Now().Add(s.Config.RefreshTokenExpiry), time.Now(), time.Now())
	if err != nil {
		return nil, "", fmt.Errorf("failed to create session: %w", err)
	}

	_, _ = s.DB.Exec(ctx, `UPDATE users SET last_login_at = NOW(), updated_at = NOW() WHERE id = $1`, user.ID)

	resp := &AuthResponse{
		AccessToken: accessToken,
		ExpiresIn:   int(s.Config.AccessTokenExpiry.Seconds()),
		User:        user.ToResponse(),
	}

	return resp, refreshToken, nil
}

func (s *Service) getUserByID(ctx context.Context, id string) (*User, error) {
	var u User
	err := s.DB.QueryRow(ctx, `
		SELECT id, email, whatsapp, password_hash, email_verified, whatsapp_verified, status, last_login_at, last_login_ip, created_at, updated_at
		FROM users WHERE id = $1
	`, id).Scan(&u.ID, &u.Email, &u.WhatsApp, &u.PasswordHash, &u.EmailVerified, &u.WhatsAppVerified, &u.Status, &u.LastLoginAt, &u.LastLoginIP, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (s *Service) getUserByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	err := s.DB.QueryRow(ctx, `
		SELECT id, email, whatsapp, password_hash, email_verified, whatsapp_verified, status, last_login_at, last_login_ip, created_at, updated_at
		FROM users WHERE email = $1
	`, email).Scan(&u.ID, &u.Email, &u.WhatsApp, &u.PasswordHash, &u.EmailVerified, &u.WhatsAppVerified, &u.Status, &u.LastLoginAt, &u.LastLoginIP, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (s *Service) getUserByWhatsApp(ctx context.Context, whatsapp string) (*User, error) {
	var u User
	err := s.DB.QueryRow(ctx, `
		SELECT id, email, whatsapp, password_hash, email_verified, whatsapp_verified, status, last_login_at, last_login_ip, created_at, updated_at
		FROM users WHERE whatsapp = $1
	`, whatsapp).Scan(&u.ID, &u.Email, &u.WhatsApp, &u.PasswordHash, &u.EmailVerified, &u.WhatsAppVerified, &u.Status, &u.LastLoginAt, &u.LastLoginIP, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (s *Service) storeOTP(ctx context.Context, identifier, identifierType, purpose, code string) error {
	id := shared.GenerateUUID()
	_, err := s.DB.Exec(ctx, `
		INSERT INTO otp_codes (id, identifier, identifier_type, code, purpose, attempts, max_attempts, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, 0, $6, $7, $8)
	`, id, identifier, identifierType, code, purpose, s.Config.OTPMaxAttempts, time.Now().Add(s.Config.OTPExpiry), time.Now())
	return err
}

func (s *Service) validateOTP(ctx context.Context, identifier, code, purpose string) error {
	var otp OTPRecord
	err := s.DB.QueryRow(ctx, `
		SELECT id, code, attempts, max_attempts, expires_at
		FROM otp_codes
		WHERE identifier = $1 AND purpose = $2 AND used_at IS NULL AND expires_at > NOW()
		ORDER BY created_at DESC LIMIT 1
	`, identifier, purpose).Scan(&otp.ID, &otp.Code, &otp.Attempts, &otp.MaxAttempts, &otp.ExpiresAt)
	if err != nil {
		return fmt.Errorf("invalid or expired OTP")
	}

	if otp.Attempts >= otp.MaxAttempts {
		return fmt.Errorf("too many failed attempts")
	}

	if otp.Code != code {
		_, _ = s.DB.Exec(ctx, `UPDATE otp_codes SET attempts = attempts + 1 WHERE id = $1`, otp.ID)
		return fmt.Errorf("invalid OTP")
	}

	_, _ = s.DB.Exec(ctx, `UPDATE otp_codes SET used_at = NOW() WHERE id = $1`, otp.ID)
	return nil
}

func (s *Service) blacklistToken(ctx context.Context, jti, userID string, expiry time.Duration) error {
	_, err := s.DB.Exec(ctx, `
		INSERT INTO blacklisted_tokens (id, token_jti, user_id, reason, expires_at, created_at)
		VALUES ($1, $2, $3, 'logout', $4, $5)
	`, shared.GenerateUUID(), jti, userID, time.Now().Add(expiry), time.Now())
	return err
}
