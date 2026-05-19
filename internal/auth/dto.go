package auth

import "time"

// --- Request DTOs ---

type RegisterRequest struct {
	Email    *string `json:"email,omitempty"`
	WhatsApp *string `json:"whatsapp,omitempty"`
	Password string  `json:"password"`
}

type LoginEmailRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginOTPRequest struct {
	WhatsApp string `json:"whatsapp"`
	OTP      string `json:"otp"`
}

type RequestOTPRequest struct {
	WhatsApp string `json:"whatsapp"`
	Purpose  string `json:"purpose"` // "login", "verify_whatsapp"
}

type RefreshTokenRequest struct {
	// Refresh token comes from HttpOnly cookie, not body
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

type ResetPasswordRequest struct {
	Identifier  string `json:"identifier"` // email or whatsapp
	OTP         string `json:"otp"`
	NewPassword string `json:"new_password"`
}

type VerifyOTPRequest struct {
	Identifier string `json:"identifier"`
	OTP        string `json:"otp"`
	Purpose    string `json:"purpose"`
}

// --- Response DTOs ---

type AuthResponse struct {
	AccessToken string       `json:"access_token"`
	ExpiresIn   int          `json:"expires_in"`
	User        UserResponse `json:"user"`
}

type UserResponse struct {
	ID               string  `json:"id"`
	Email            *string `json:"email,omitempty"`
	WhatsApp         *string `json:"whatsapp,omitempty"`
	EmailVerified    bool    `json:"email_verified"`
	WhatsAppVerified bool    `json:"whatsapp_verified"`
	Status           string  `json:"status"`
	CreatedAt        string  `json:"created_at"`
}

type SessionResponse struct {
	ID         string  `json:"id"`
	DeviceName *string `json:"device_name,omitempty"`
	IPAddress  *string `json:"ip_address,omitempty"`
	LastUsedAt string  `json:"last_used_at"`
	CreatedAt  string  `json:"created_at"`
}

type MessageResponse struct {
	Message string `json:"message"`
}

// --- Domain Models ---

type User struct {
	ID               string
	Email            *string
	WhatsApp         *string
	PasswordHash     *string
	EmailVerified    bool
	WhatsAppVerified bool
	Status           string
	LastLoginAt      *time.Time
	LastLoginIP      *string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type Session struct {
	ID           string
	UserID       string
	RefreshToken string
	DeviceName   *string
	DeviceType   *string
	IPAddress    *string
	UserAgent    *string
	ExpiresAt    time.Time
	LastUsedAt   time.Time
	CreatedAt    time.Time
}

type OTPRecord struct {
	ID             string
	Identifier     string
	IdentifierType string
	Code           string
	Purpose        string
	Attempts       int
	MaxAttempts    int
	ExpiresAt      time.Time
	UsedAt         *time.Time
	CreatedAt      time.Time
}

// --- Constants ---

const (
	StatusActive     = "ACTIVE"
	StatusSuspended  = "SUSPENDED"
	StatusUnverified = "UNVERIFIED"
	StatusDeleted    = "DELETED"

	IdentifierTypeEmail = "email"
	IdentifierTypeWA    = "whatsapp"

	OTPPurposeLogin     = "login"
	OTPPurposeRegister  = "register"
	OTPPurposeResetPass = "reset_password"
	OTPPurposeVerifyWA  = "verify_whatsapp"
	OTPPurposeVerifyEmail = "verify_email"
)

func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:               u.ID,
		Email:            u.Email,
		WhatsApp:         u.WhatsApp,
		EmailVerified:    u.EmailVerified,
		WhatsAppVerified: u.WhatsAppVerified,
		Status:           u.Status,
		CreatedAt:        u.CreatedAt.Format(time.RFC3339),
	}
}

func (s *Session) ToResponse() SessionResponse {
	return SessionResponse{
		ID:         s.ID,
		DeviceName: s.DeviceName,
		IPAddress:  s.IPAddress,
		LastUsedAt: s.LastUsedAt.Format(time.RFC3339),
		CreatedAt:  s.CreatedAt.Format(time.RFC3339),
	}
}
