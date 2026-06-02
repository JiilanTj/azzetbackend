package auth

import (
	"time"

	"codeberg.org/azzet/azzetbe/internal/db"
)

// --- Request DTOs ---

// RegisterRequest represents the registration payload
// @Description Registration request body. Either email or whatsapp is required. Password is always required.
type RegisterRequest struct {
	Name     string  `json:"name" example:"John Doe"`
	Email    *string `json:"email,omitempty" example:"user@example.com"`
	WhatsApp *string `json:"whatsapp,omitempty" example:"+628123456789"`
	Password string  `json:"password" example:"SecurePass123"`
}

// LoginEmailRequest represents email login payload
// @Description Login with email and password
type LoginEmailRequest struct {
	Email    string `json:"email" example:"user@example.com"`
	Password string `json:"password" example:"SecurePass123"`
}

// LoginOTPRequest represents WhatsApp OTP login payload
// @Description Login with WhatsApp number and OTP code
type LoginOTPRequest struct {
	WhatsApp string `json:"whatsapp" example:"+628123456789"`
	OTP      string `json:"otp" example:"123456"`
}

// RequestOTPRequest represents OTP request payload
// @Description Request an OTP code to be sent via WhatsApp
type RequestOTPRequest struct {
	WhatsApp string `json:"whatsapp" example:"+628123456789"`
	Purpose  string `json:"purpose" example:"login" enums:"login,verify_whatsapp,reset_password"`
}

// ChangePasswordRequest represents password change payload
// @Description Change password (requires authentication)
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" example:"OldPass123"`
	NewPassword string `json:"new_password" example:"NewSecurePass456"`
}

// ResetPasswordRequest represents password reset payload
// @Description Reset password using OTP verification
type ResetPasswordRequest struct {
	Identifier  string `json:"identifier" example:"+628123456789"`
	OTP         string `json:"otp" example:"123456"`
	NewPassword string `json:"new_password" example:"NewSecurePass456"`
}

// VerifyOTPRequest represents OTP verification payload
// @Description Verify OTP to activate account. Password required when activating an unverified account.
type VerifyOTPRequest struct {
	Identifier string `json:"identifier" example:"+628123456789"`
	OTP        string `json:"otp" example:"123456"`
	Purpose    string `json:"purpose" example:"verify_whatsapp" enums:"verify_whatsapp,verify_email"`
	Password   string `json:"password,omitempty" example:"SecurePass123"`
}

// --- Response DTOs ---

// AuthResponse represents the authentication response
// @Description Authentication response with access token and user info. Refresh token is set as HttpOnly cookie.
type AuthResponse struct {
	AccessToken string       `json:"access_token" example:"eyJhbGciOiJIUzI1NiIs..."`
	ExpiresIn   int          `json:"expires_in" example:"900"`
	User        UserResponse `json:"user"`
}

// RegisterResponse represents the registration response
// @Description Registration response with user info and verification message
type RegisterResponse struct {
	User    UserResponse `json:"user"`
	Message string       `json:"message" example:"Registration successful. Please verify your account."`
}

// UserResponse represents user profile data
// @Description User profile information
type UserResponse struct {
	ID               string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name             *string `json:"name,omitempty" example:"John Doe"`
	Email            *string `json:"email,omitempty" example:"user@example.com"`
	WhatsApp         *string `json:"whatsapp,omitempty" example:"+628123456789"`
	EmailVerified    bool    `json:"email_verified" example:"true"`
	WhatsAppVerified bool    `json:"whatsapp_verified" example:"false"`
	Status           string  `json:"status" example:"ACTIVE" enums:"ACTIVE,SUSPENDED,UNVERIFIED,DELETED"`
	CreatedAt        string  `json:"created_at" example:"2026-05-19T10:00:00Z"`
}

// SessionResponse represents an active session
// @Description Active session information
type SessionResponse struct {
	ID         string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	DeviceName *string `json:"device_name,omitempty" example:"Chrome on MacOS"`
	IPAddress  *string `json:"ip_address,omitempty" example:"192.168.1.1"`
	LastUsedAt string  `json:"last_used_at" example:"2026-05-19T10:00:00Z"`
	CreatedAt  string  `json:"created_at" example:"2026-05-19T09:00:00Z"`
}

// MessageResponse represents a simple message response
// @Description Simple message response
type MessageResponse struct {
	Message string `json:"message" example:"Operation successful"`
}

// --- Constants ---

const (
	StatusActive     = "ACTIVE"
	StatusSuspended  = "SUSPENDED"
	StatusUnverified = "UNVERIFIED"
	StatusDeleted    = "DELETED"

	IdentifierTypeEmail = "email"
	IdentifierTypeWA    = "whatsapp"

	OTPPurposeLogin       = "login"
	OTPPurposeRegister    = "register"
	OTPPurposeResetPass   = "reset_password"
	OTPPurposeVerifyWA    = "verify_whatsapp"
	OTPPurposeVerifyEmail = "verify_email"
)

// --- Converters ---

func SessionToResponse(s *db.Session) SessionResponse {
	var deviceName, ipAddress *string
	if s.DeviceName.Valid {
		deviceName = &s.DeviceName.String
	}
	if s.IpAddress != nil {
		addr := s.IpAddress.String()
		ipAddress = &addr
	}

	return SessionResponse{
		ID:         s.ID.String(),
		DeviceName: deviceName,
		IPAddress:  ipAddress,
		LastUsedAt: s.LastUsedAt.Format(time.RFC3339),
		CreatedAt:  s.CreatedAt.Format(time.RFC3339),
	}
}
