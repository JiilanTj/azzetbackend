package auth

import (
	"time"

	"codeberg.org/azzet/azzetbe/internal/db"
)

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
