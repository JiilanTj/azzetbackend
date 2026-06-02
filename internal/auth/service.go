package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/netip"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"codeberg.org/azzet/azzetbe/internal/db"
	"codeberg.org/azzet/azzetbe/internal/entity"
	"codeberg.org/azzet/azzetbe/internal/events"
	rdb "codeberg.org/azzet/azzetbe/internal/redis"
	"codeberg.org/azzet/azzetbe/internal/shared"
	"codeberg.org/azzet/azzetbe/internal/subscription"
	"codeberg.org/azzet/azzetbe/internal/workspace"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUserNotFound = errors.New("user not found")

type Service struct {
	Queries             *db.Queries
	Pool                *pgxpool.Pool
	Redis               *rdb.Redis
	JWT                 *shared.JWTService
	OTP                 *shared.OTPService
	Zenziva             *shared.ZenzivaClient
	Email               *shared.EmailOTPSender
	Config              *ServiceConfig
	EntityService       *entity.Service
	WorkspaceService    *workspace.Service
	SubscriptionService *subscription.Service
}

type ServiceConfig struct {
	AccessTokenExpiry  time.Duration
	RefreshTokenExpiry time.Duration
	OTPExpiry          time.Duration
	OTPMaxAttempts     int
}

// LoginContext carries request metadata for session tracking
type LoginContext struct {
	IPAddress string
	UserAgent string
	DeviceName string
}

func NewService(queries *db.Queries, pool *pgxpool.Pool, redis *rdb.Redis, jwt *shared.JWTService, otp *shared.OTPService, zenziva *shared.ZenzivaClient, email *shared.EmailOTPSender, cfg *ServiceConfig) *Service {
	return &Service{
		Queries: queries,
		Pool:    pool,
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

	// Check duplicates (generic error to prevent enumeration)
	if req.Email != nil && *req.Email != "" {
		exists, err := s.Queries.ExistsByEmail(ctx, pgtype.Text{String: *req.Email, Valid: true})
		if err != nil {
			return nil, fmt.Errorf("registration failed")
		}
		if exists {
			return nil, fmt.Errorf("registration failed")
		}
	}
	if req.WhatsApp != nil && *req.WhatsApp != "" {
		exists, err := s.Queries.ExistsByWhatsApp(ctx, pgtype.Text{String: *req.WhatsApp, Valid: true})
		if err != nil {
			return nil, fmt.Errorf("registration failed")
		}
		if exists {
			return nil, fmt.Errorf("registration failed")
		}
	}

	hash, err := shared.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("registration failed")
	}

	now := time.Now()
	params := db.CreateUserParams{
		ID:           uuid.New(),
		Email:        toPgText(req.Email),
		Whatsapp:     toPgText(req.WhatsApp),
		PasswordHash: pgtype.Text{String: hash, Valid: true},
		Name:         toPgText(&req.Name),
		Status:       StatusUnverified,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	user, err := s.Queries.CreateUser(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("registration failed")
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

	// Create personal entity + workspace synchronously (instant, no polling needed)
	// Also emit event for audit trail and future consumers
	name := req.Name
	if name == "" {
		name = "Personal"
	}
	personalEntity, err := s.EntityService.CreatePersonalEntity(ctx, user.ID, name)
	if err == nil {
		_ = s.WorkspaceService.CreatePersonalWorkspace(ctx, personalEntity.ID, user.ID)

		// Auto-assign free plan to personal workspace (if a free plan exists)
		if s.SubscriptionService != nil {
			freePlan, err := s.Queries.GetFreePlan(ctx)
			if err == nil {
				if _, subErr := s.SubscriptionService.Subscribe(ctx, personalEntity.ID.String(), &subscription.SubscribeRequest{
					PlanID: freePlan.ID.String(),
				}); subErr != nil {
					slog.Warn("failed to assign free plan on registration",
						"user_id", user.ID.String(),
						"entity_id", personalEntity.ID.String(),
						"error", subErr,
					)
				}
			}
		}
	}

	// Emit user.registered event (for audit, notifications, future consumers)
	_ = events.EmitEventDirect(ctx, s.Pool, events.UserRegistered, map[string]string{
		"user_id": user.ID.String(),
		"name":    req.Name,
	})

	return &user, nil
}

// LoginWithEmail authenticates user with email + password
func (s *Service) LoginWithEmail(ctx context.Context, req *LoginEmailRequest, lc *LoginContext) (*AuthResponse, string, error) {
	user, err := s.Queries.GetUserByEmail(ctx, pgtype.Text{String: req.Email, Valid: true})
	if err != nil {
		return nil, "", fmt.Errorf("invalid credentials")
	}

	if !user.PasswordHash.Valid || !shared.VerifyPassword(user.PasswordHash.String, req.Password) {
		return nil, "", fmt.Errorf("invalid credentials")
	}
	if user.Status == StatusSuspended {
		return nil, "", fmt.Errorf("account is suspended")
	}
	if user.Status != StatusActive {
		return nil, "", fmt.Errorf("account is not verified")
	}

	return s.createTokenPair(ctx, &user, lc)
}

// LoginWithOTP authenticates user with whatsapp + OTP code
func (s *Service) LoginWithOTP(ctx context.Context, req *LoginOTPRequest, lc *LoginContext) (*AuthResponse, string, error) {
	if err := s.validateOTP(ctx, req.WhatsApp, req.OTP, OTPPurposeLogin); err != nil {
		return nil, "", err
	}

	user, err := s.Queries.GetUserByWhatsApp(ctx, pgtype.Text{String: req.WhatsApp, Valid: true})
	if err != nil {
		return nil, "", fmt.Errorf("invalid credentials")
	}
	if user.Status == StatusSuspended {
		return nil, "", fmt.Errorf("account is suspended")
	}
	if user.Status != StatusActive {
		return nil, "", fmt.Errorf("account is not verified")
	}

	return s.createTokenPair(ctx, &user, lc)
}

// RequestOTP sends an OTP to the user's whatsapp
func (s *Service) RequestOTP(ctx context.Context, req *RequestOTPRequest) error {
	// Validate purpose
	if req.Purpose != OTPPurposeLogin && req.Purpose != OTPPurposeVerifyWA && req.Purpose != OTPPurposeResetPass {
		return fmt.Errorf("invalid OTP purpose")
	}

	if err := s.checkOTPRateLimit(ctx, req.WhatsApp); err != nil {
		return err
	}

	_, err := s.Queries.GetUserByWhatsApp(ctx, pgtype.Text{String: req.WhatsApp, Valid: true})
	if err != nil {
		// Generic error to prevent enumeration
		return fmt.Errorf("OTP request failed")
	}

	code := s.OTP.Generate()
	if err := s.storeOTP(ctx, req.WhatsApp, IdentifierTypeWA, req.Purpose, code); err != nil {
		return fmt.Errorf("OTP request failed")
	}

	return s.Zenziva.SendOTP(ctx, req.WhatsApp, code)
}

// RefreshToken rotates the refresh token and issues new access token
func (s *Service) RefreshToken(ctx context.Context, refreshToken string, lc *LoginContext) (*AuthResponse, string, error) {
	claims, err := s.JWT.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, "", fmt.Errorf("invalid refresh token")
	}

	sessionID, err := uuid.Parse(claims.SessionID)
	if err != nil {
		return nil, "", fmt.Errorf("invalid session")
	}

	// Verify session by hashed refresh token
	session, err := s.Queries.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, "", fmt.Errorf("session not found")
	}
	if time.Now().After(session.ExpiresAt) {
		return nil, "", fmt.Errorf("session expired")
	}
	if session.RefreshToken != hashToken(refreshToken) {
		return nil, "", fmt.Errorf("invalid refresh token")
	}

	// Get user
	user, err := s.Queries.GetUserByID(ctx, session.UserID)
	if err != nil {
		return nil, "", fmt.Errorf("user not found")
	}
	if user.Status == StatusSuspended {
		_ = s.Queries.DeleteSessionByID(ctx, sessionID)
		return nil, "", fmt.Errorf("account is suspended")
	}
	if user.Status != StatusActive {
		_ = s.Queries.DeleteSessionByID(ctx, sessionID)
		return nil, "", fmt.Errorf("account is not verified")
	}

	resp, newRefresh, err := s.createTokenPair(ctx, &user, lc)
	if err != nil {
		return nil, "", err
	}

	if err := s.Queries.DeleteSessionByID(ctx, sessionID); err != nil {
		slog.Warn("failed to delete rotated session", "session_id", sessionID.String(), "error", err)
	}

	return resp, newRefresh, nil
}

// Logout blacklists the access token (Redis) and deletes the session
func (s *Service) Logout(ctx context.Context, accessToken, refreshToken string) error {
	claims, err := s.JWT.ValidateAccessToken(accessToken)
	if err != nil {
		return fmt.Errorf("invalid access token")
	}

	// Blacklist access token in Redis (auto-expires)
	if err := s.Redis.Set(ctx, "blacklist:"+claims.JTI, claims.UserID, s.Config.AccessTokenExpiry).Err(); err != nil {
		return fmt.Errorf("failed to revoke token")
	}

	// Delete session by hashed refresh token
	if refreshToken != "" {
		_ = s.Queries.DeleteSessionByRefreshToken(ctx, hashToken(refreshToken))
	}

	return nil
}

// LogoutAll blacklists current token and deletes all user sessions
func (s *Service) LogoutAll(ctx context.Context, accessToken string) error {
	claims, err := s.JWT.ValidateAccessToken(accessToken)
	if err != nil {
		return fmt.Errorf("invalid access token")
	}

	if err := s.Redis.Set(ctx, "blacklist:"+claims.JTI, claims.UserID, s.Config.AccessTokenExpiry).Err(); err != nil {
		return fmt.Errorf("failed to revoke token")
	}

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
		return nil, ErrUserNotFound
	}
	user, err := s.Queries.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
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
	// Validate purpose
	if req.Purpose != OTPPurposeVerifyWA && req.Purpose != OTPPurposeVerifyEmail {
		return fmt.Errorf("invalid verification purpose")
	}

	if err := s.validateOTP(ctx, req.Identifier, req.OTP, req.Purpose); err != nil {
		return err
	}

	var user db.User
	var err error
	if req.Purpose == OTPPurposeVerifyEmail {
		user, err = s.Queries.GetUserByEmail(ctx, pgtype.Text{String: req.Identifier, Valid: true})
	} else {
		user, err = s.Queries.GetUserByWhatsApp(ctx, pgtype.Text{String: req.Identifier, Valid: true})
	}
	if err != nil {
		return fmt.Errorf("user not found")
	}
	if user.Status == StatusUnverified {
		if req.Password == "" {
			return fmt.Errorf("password is required to activate account")
		}
		if !user.PasswordHash.Valid || !shared.VerifyPassword(user.PasswordHash.String, req.Password) {
			return fmt.Errorf("invalid password")
		}
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
		return ErrUserNotFound
	}

	user, err := s.Queries.GetUserByID(ctx, id)
	if err != nil {
		return ErrUserNotFound
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

	if err := s.Queries.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
		ID:           id,
		PasswordHash: pgtype.Text{String: hash, Valid: true},
	}); err != nil {
		return err
	}
	_ = s.Queries.DeleteUserSessions(ctx, id)
	return nil
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

	user, lookupErr := s.Queries.GetUserByEmail(ctx, pgtype.Text{String: req.Identifier, Valid: true})
	if lookupErr != nil {
		user, lookupErr = s.Queries.GetUserByWhatsApp(ctx, pgtype.Text{String: req.Identifier, Valid: true})
		if lookupErr != nil {
			return fmt.Errorf("user not found")
		}
	}

	if err := s.Queries.ResetPasswordByIdentifier(ctx, db.ResetPasswordByIdentifierParams{
		Email:        pgtype.Text{String: req.Identifier, Valid: true},
		PasswordHash: pgtype.Text{String: hash, Valid: true},
	}); err != nil {
		return err
	}
	_ = s.Queries.DeleteUserSessions(ctx, user.ID)
	return nil
}

// IsTokenBlacklisted checks Redis for blacklisted token
func (s *Service) IsTokenBlacklisted(ctx context.Context, jti string) (bool, error) {
	return s.Redis.Exists(ctx, "blacklist:"+jti)
}

// --- Private helpers ---

func (s *Service) createTokenPair(ctx context.Context, user *db.User, lc *LoginContext) (*AuthResponse, string, error) {
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
	sessionParams := db.CreateSessionParams{
		ID:           sessionID,
		UserID:       user.ID,
		RefreshToken: hashToken(refreshToken), // Store hash, not plaintext
		ExpiresAt:    now.Add(s.Config.RefreshTokenExpiry),
		LastUsedAt:   now,
		CreatedAt:    now,
	}

	// Populate device/IP info if available
	if lc != nil {
		if lc.IPAddress != "" {
			if addr, err := netip.ParseAddr(lc.IPAddress); err == nil {
				sessionParams.IpAddress = &addr
			}
		}
		if lc.UserAgent != "" {
			sessionParams.UserAgent = pgtype.Text{String: lc.UserAgent, Valid: true}
		}
		if lc.DeviceName != "" {
			sessionParams.DeviceName = pgtype.Text{String: lc.DeviceName, Valid: true}
		}
	}

	_, err = s.Queries.CreateSession(ctx, sessionParams)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create session: %w", err)
	}

	// Update last login with IP
	loginParams := db.UpdateUserLastLoginParams{ID: user.ID}
	if lc != nil && lc.IPAddress != "" {
		if addr, err := netip.ParseAddr(lc.IPAddress); err == nil {
			loginParams.LastLoginIp = &addr
		}
	}
	_ = s.Queries.UpdateUserLastLogin(ctx, loginParams)

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
		Code:           shared.HashOTP(code), // Store hash, not plaintext
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
		return fmt.Errorf("invalid or expired OTP")
	}

	if otp.Attempts >= otp.MaxAttempts {
		return fmt.Errorf("too many failed attempts")
	}

	// Compare hashed OTP
	if !shared.VerifyOTP(code, otp.Code) {
		if err := s.Queries.IncrementOTPAttempts(ctx, otp.ID); err != nil {
			slog.Warn("failed to increment OTP attempts", "otp_id", otp.ID.String(), "error", err)
		}
		return fmt.Errorf("invalid OTP")
	}

	if err := s.Queries.MarkOTPUsed(ctx, otp.ID); err != nil {
		return fmt.Errorf("failed to mark OTP used: %w", err)
	}
	return nil
}

func (s *Service) checkOTPRateLimit(ctx context.Context, identifier string) error {
	key := "otp:ratelimit:" + identifier
	count, err := s.Redis.Incr(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("OTP request failed")
	}
	if count == 1 {
		_ = s.Redis.Expire(ctx, key, 15*time.Minute).Err()
	}
	if count > 5 {
		return fmt.Errorf("too many OTP requests, try again later")
	}
	return nil
}

// hashToken creates a SHA256 hash of a token for secure storage
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// --- Helpers ---

func toPgText(s *string) pgtype.Text {
	if s == nil || *s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func UserToResponse(u *db.User) UserResponse {
	var name, email, whatsapp *string
	if u.Name.Valid {
		name = &u.Name.String
	}
	if u.Email.Valid {
		email = &u.Email.String
	}
	if u.Whatsapp.Valid {
		whatsapp = &u.Whatsapp.String
	}

	return UserResponse{
		ID:               u.ID.String(),
		Name:             name,
		Email:            email,
		WhatsApp:         whatsapp,
		EmailVerified:    u.EmailVerified,
		WhatsAppVerified: u.WhatsappVerified,
		Status:           u.Status,
		CreatedAt:        u.CreatedAt.Format(time.RFC3339),
	}
}
