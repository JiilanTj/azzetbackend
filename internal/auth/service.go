package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"codeberg.org/azzet/azzetbe/internal/db"
	rdb "codeberg.org/azzet/azzetbe/internal/redis"
	"codeberg.org/azzet/azzetbe/internal/shared"
)

type Service struct {
	Queries *db.Queries
	Redis   *rdb.Redis
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

func NewService(queries *db.Queries, redis *rdb.Redis, jwt *shared.JWTService, otp *shared.OTPService, zenziva *shared.ZenzivaClient, email *shared.EmailOTPSender, cfg *ServiceConfig) *Service {
	return &Service{
		Queries: queries,
		Redis:   redis,
		JWT:     jwt,
		OTP:     otp,
		Zenziva: zenziva,
		Email:   email,
		Config:  cfg,
	}
}

// Register creates a new user with email or whatsapp + password (always required)
func (s *Service) Register(ctx context.Context, req *RegisterRequest) (*db.User, error) {
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
		exists, err := s.Queries.ExistsByEmail(ctx, pgtype.Text{String: *req.Email, Valid: true})
		if err != nil {
			return nil, fmt.Errorf("failed to check email: %w", err)
		}
		if exists {
			return nil, fmt.Errorf("email already registered")
		}
	}
	if req.WhatsApp != nil && *req.WhatsApp != "" {
		exists, err := s.Queries.ExistsByWhatsApp(ctx, pgtype.Text{String: *req.WhatsApp, Valid: true})
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

	now := time.Now()
	params := db.CreateUserParams{
		ID:           uuid.New(),
		Email:        toPgText(req.Email),
		Whatsapp:     toPgText(req.WhatsApp),
		PasswordHash: pgtype.Text{String: hash, Valid: true},
		Status:       StatusUnverified,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	user, err := s.Queries.CreateUser(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
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

	return &user, nil
}

// LoginWithEmail authenticates user with email + password
func (s *Service) LoginWithEmail(ctx context.Context, req *LoginEmailRequest) (*AuthResponse, string, error) {
	user, err := s.Queries.GetUserByEmail(ctx, pgtype.Text{String: req.Email, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", fmt.Errorf("invalid credentials")
		}
		return nil, "", err
	}

	if !user.PasswordHash.Valid || !shared.VerifyPassword(user.PasswordHash.String, req.Password) {
		return nil, "", fmt.Errorf("invalid credentials")
	}
	if user.Status == StatusSuspended {
		return nil, "", fmt.Errorf("account is suspended")
	}

	return s.createTokenPair(ctx, &user)
}

// LoginWithOTP authenticates user with whatsapp + OTP code
func (s *Service) LoginWithOTP(ctx context.Context, req *LoginOTPRequest) (*AuthResponse, string, error) {
	if err := s.validateOTP(ctx, req.WhatsApp, req.OTP, OTPPurposeLogin); err != nil {
		return nil, "", err
	}

	user, err := s.Queries.GetUserByWhatsApp(ctx, pgtype.Text{String: req.WhatsApp, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", fmt.Errorf("user not found")
		}
		return nil, "", err
	}
	if user.Status == StatusSuspended {
		return nil, "", fmt.Errorf("account is suspended")
	}

	return s.createTokenPair(ctx, &user)
}

// RequestOTP sends an OTP to the user's whatsapp
func (s *Service) RequestOTP(ctx context.Context, req *RequestOTPRequest) error {
	_, err := s.Queries.GetUserByWhatsApp(ctx, pgtype.Text{String: req.WhatsApp, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("user not found")
		}
		return err
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

	sessionID, err := uuid.Parse(claims.SessionID)
	if err != nil {
		return nil, "", fmt.Errorf("invalid session")
	}

	// Verify session
	session, err := s.Queries.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, "", fmt.Errorf("session not found")
	}
	if time.Now().After(session.ExpiresAt) {
		return nil, "", fmt.Errorf("session expired")
	}
	if session.RefreshToken != refreshToken {
		return nil, "", fmt.Errorf("invalid refresh token")
	}

	// Get user
	user, err := s.Queries.GetUserByID(ctx, session.UserID)
	if err != nil {
		return nil, "", fmt.Errorf("user not found")
	}

	// Delete old session
	_ = s.Queries.DeleteSessionByID(ctx, sessionID)

	// Create new token pair
	return s.createTokenPair(ctx, &user)
}

// Logout blacklists the access token (Redis) and deletes the session
func (s *Service) Logout(ctx context.Context, accessToken, refreshToken string) error {
	claims, err := s.JWT.ValidateAccessToken(accessToken)
	if err != nil {
		return fmt.Errorf("invalid access token")
	}

	// Blacklist access token in Redis (auto-expires)
	_ = s.Redis.Set(ctx, "blacklist:"+claims.JTI, claims.UserID, s.Config.AccessTokenExpiry)

	// Delete session
	if refreshToken != "" {
		_ = s.Queries.DeleteSessionByRefreshToken(ctx, refreshToken)
	}

	return nil
}

// LogoutAll blacklists current token and deletes all user sessions
func (s *Service) LogoutAll(ctx context.Context, accessToken string) error {
	claims, err := s.JWT.ValidateAccessToken(accessToken)
	if err != nil {
		return fmt.Errorf("invalid access token")
	}

	// Blacklist access token in Redis
	_ = s.Redis.Set(ctx, "blacklist:"+claims.JTI, claims.UserID, s.Config.AccessTokenExpiry)

	// Delete all sessions
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return err
	}
	_ = s.Queries.DeleteUserSessions(ctx, userID)

	return nil
}

// GetMe returns the current user
func (s *Service) GetMe(ctx context.Context, userID string) (*db.User, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id")
	}
	user, err := s.Queries.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// GetSessions returns all active sessions for a user
func (s *Service) GetSessions(ctx context.Context, userID string) ([]db.Session, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id")
	}
	return s.Queries.GetUserSessions(ctx, id)
}

// RevokeSession deletes a specific session
func (s *Service) RevokeSession(ctx context.Context, userID, sessionID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user id")
	}
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return fmt.Errorf("invalid session id")
	}
	return s.Queries.DeleteSession(ctx, db.DeleteSessionParams{
		ID:     sid,
		UserID: uid,
	})
}

// VerifyOTP verifies an OTP and activates the user
func (s *Service) VerifyOTP(ctx context.Context, req *VerifyOTPRequest) error {
	if err := s.validateOTP(ctx, req.Identifier, req.OTP, req.Purpose); err != nil {
		return err
	}

	switch req.Purpose {
	case OTPPurposeVerifyWA:
		return s.Queries.VerifyUserWhatsApp(ctx, pgtype.Text{String: req.Identifier, Valid: true})
	case OTPPurposeVerifyEmail:
		return s.Queries.VerifyUserEmail(ctx, pgtype.Text{String: req.Identifier, Valid: true})
	}
	return nil
}

// ChangePassword changes the user's password
func (s *Service) ChangePassword(ctx context.Context, userID string, req *ChangePasswordRequest) error {
	id, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user id")
	}

	user, err := s.Queries.GetUserByID(ctx, id)
	if err != nil {
		return fmt.Errorf("user not found")
	}
	if !user.PasswordHash.Valid || !shared.VerifyPassword(user.PasswordHash.String, req.OldPassword) {
		return fmt.Errorf("invalid old password")
	}
	if len(req.NewPassword) < 8 {
		return fmt.Errorf("new password must be at least 8 characters")
	}

	hash, err := shared.HashPassword(req.NewPassword)
	if err != nil {
		return err
	}

	return s.Queries.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
		ID:           id,
		PasswordHash: pgtype.Text{String: hash, Valid: true},
	})
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

	return s.Queries.ResetPasswordByIdentifier(ctx, db.ResetPasswordByIdentifierParams{
		Email:        pgtype.Text{String: req.Identifier, Valid: true},
		PasswordHash: pgtype.Text{String: hash, Valid: true},
	})
}

// IsTokenBlacklisted checks Redis for blacklisted token
func (s *Service) IsTokenBlacklisted(ctx context.Context, jti string) (bool, error) {
	return s.Redis.Exists(ctx, "blacklist:"+jti)
}

// --- Private helpers ---

func (s *Service) createTokenPair(ctx context.Context, user *db.User) (*AuthResponse, string, error) {
	sessionID := uuid.New()

	accessToken, err := s.JWT.GenerateAccessToken(user.ID.String(), sessionID.String())
	if err != nil {
		return nil, "", err
	}

	refreshToken, err := s.JWT.GenerateRefreshToken(user.ID.String(), sessionID.String())
	if err != nil {
		return nil, "", err
	}

	now := time.Now()
	_, err = s.Queries.CreateSession(ctx, db.CreateSessionParams{
		ID:           sessionID,
		UserID:       user.ID,
		RefreshToken: refreshToken,
		ExpiresAt:    now.Add(s.Config.RefreshTokenExpiry),
		LastUsedAt:   now,
		CreatedAt:    now,
	})
	if err != nil {
		return nil, "", fmt.Errorf("failed to create session: %w", err)
	}

	// Update last login
	_ = s.Queries.UpdateUserLastLogin(ctx, db.UpdateUserLastLoginParams{
		ID: user.ID,
	})

	resp := &AuthResponse{
		AccessToken: accessToken,
		ExpiresIn:   int(s.Config.AccessTokenExpiry.Seconds()),
		User:        UserToResponse(user),
	}

	return resp, refreshToken, nil
}

func (s *Service) storeOTP(ctx context.Context, identifier, identifierType, purpose, code string) error {
	return s.Queries.CreateOTP(ctx, db.CreateOTPParams{
		ID:             uuid.New(),
		Identifier:     identifier,
		IdentifierType: identifierType,
		Code:           code,
		Purpose:        purpose,
		MaxAttempts:    int32(s.Config.OTPMaxAttempts),
		ExpiresAt:      time.Now().Add(s.Config.OTPExpiry),
		CreatedAt:      time.Now(),
	})
}

func (s *Service) validateOTP(ctx context.Context, identifier, code, purpose string) error {
	otp, err := s.Queries.GetValidOTP(ctx, db.GetValidOTPParams{
		Identifier: identifier,
		Purpose:    purpose,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("invalid or expired OTP")
		}
		return err
	}

	if otp.Attempts >= otp.MaxAttempts {
		return fmt.Errorf("too many failed attempts")
	}

	if otp.Code != code {
		_ = s.Queries.IncrementOTPAttempts(ctx, otp.ID)
		return fmt.Errorf("invalid OTP")
	}

	_ = s.Queries.MarkOTPUsed(ctx, otp.ID)
	return nil
}

// --- Helpers ---

func toPgText(s *string) pgtype.Text {
	if s == nil || *s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func UserToResponse(u *db.User) UserResponse {
	var email, whatsapp *string
	if u.Email.Valid {
		email = &u.Email.String
	}
	if u.Whatsapp.Valid {
		whatsapp = &u.Whatsapp.String
	}

	return UserResponse{
		ID:               u.ID.String(),
		Email:            email,
		WhatsApp:         whatsapp,
		EmailVerified:    u.EmailVerified,
		WhatsAppVerified: u.WhatsappVerified,
		Status:           u.Status,
		CreatedAt:        u.CreatedAt.Format(time.RFC3339),
	}
}
