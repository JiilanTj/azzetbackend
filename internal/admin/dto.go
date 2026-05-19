package admin

import "time"

// --- Request DTOs ---

// LoginRequest represents admin login payload (step 1)
// @Description Admin login with email and password
type LoginRequest struct {
	Email    string `json:"email" example:"admin@azzet.com"`
	Password string `json:"password" example:"SuperSecure123!"`
}

// MFAVerifyRequest represents MFA verification payload (step 2)
// @Description Verify TOTP code after password authentication
type MFAVerifyRequest struct {
	MFAToken string `json:"mfa_token" example:"temp-mfa-token-uuid"`
	Code     string `json:"code" example:"123456"`
}

// MFASetupConfirmRequest confirms MFA setup with first TOTP code
// @Description Confirm MFA setup by providing the first TOTP code from authenticator app
type MFASetupConfirmRequest struct {
	Code string `json:"code" example:"123456"`
}

// InviteAdminRequest represents admin invitation payload
// @Description Invite a new platform admin (SUPER_ADMIN only)
type InviteAdminRequest struct {
	Email    string `json:"email" example:"support@azzet.com"`
	Name     string `json:"name" example:"Support Agent"`
	Role     string `json:"role" example:"SUPPORT" enums:"SUPER_ADMIN,SUPPORT,REVIEWER,ENGINEER"`
	Password string `json:"password" example:"TempPass123!"`
}

// UpdateAdminRequest represents admin update payload
// @Description Update admin role or status
type UpdateAdminRequest struct {
	Name   *string `json:"name,omitempty" example:"Updated Name"`
	Role   *string `json:"role,omitempty" example:"REVIEWER" enums:"SUPER_ADMIN,SUPPORT,REVIEWER,ENGINEER"`
	Status *string `json:"status,omitempty" example:"SUSPENDED" enums:"ACTIVE,SUSPENDED"`
}

// --- Response DTOs ---

// LoginResponse represents step 1 login response
// @Description Response after password verification. If MFA enabled, requires second step.
type LoginResponse struct {
	RequiresMFA bool    `json:"requires_mfa" example:"true"`
	MFAToken    *string `json:"mfa_token,omitempty" example:"temp-mfa-token-uuid"`
	// Only set if MFA is not enabled (first login, needs setup)
	AccessToken *string       `json:"access_token,omitempty"`
	ExpiresIn   *int          `json:"expires_in,omitempty" example:"600"`
	Admin       *AdminResponse `json:"admin,omitempty"`
}

// AuthResponse represents successful authentication response
// @Description Full auth response after MFA verification
type AuthResponse struct {
	AccessToken string        `json:"access_token" example:"eyJhbGciOiJIUzI1NiIs..."`
	ExpiresIn   int           `json:"expires_in" example:"600"`
	Admin       AdminResponse `json:"admin"`
}

// MFASetupResponse represents MFA setup data
// @Description MFA setup response with QR code URL and secret
type MFASetupResponse struct {
	Secret string `json:"secret" example:"JBSWY3DPEHPK3PXP"`
	QRCode string `json:"qr_code" example:"otpauth://totp/Azzet:admin@azzet.com?secret=JBSWY3DPEHPK3PXP&issuer=Azzet"`
}

// AdminResponse represents admin profile data
// @Description Platform admin profile information
type AdminResponse struct {
	ID         string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Email      string  `json:"email" example:"admin@azzet.com"`
	Name       string  `json:"name" example:"Azzet Admin"`
	Role       string  `json:"role" example:"SUPER_ADMIN" enums:"SUPER_ADMIN,SUPPORT,REVIEWER,ENGINEER"`
	MFAEnabled bool    `json:"mfa_enabled" example:"true"`
	Status     string  `json:"status" example:"ACTIVE" enums:"ACTIVE,SUSPENDED,DELETED"`
	LastLogin  *string `json:"last_login,omitempty" example:"2026-05-19T10:00:00Z"`
	CreatedAt  string  `json:"created_at" example:"2026-05-19T09:00:00Z"`
}

// MessageResponse represents a simple message
// @Description Simple message response
type MessageResponse struct {
	Message string `json:"message" example:"Operation successful"`
}

// --- Constants ---

const (
	RoleSuperAdmin = "SUPER_ADMIN"
	RoleSupport    = "SUPPORT"
	RoleReviewer   = "REVIEWER"
	RoleEngineer   = "ENGINEER"

	AdminStatusActive    = "ACTIVE"
	AdminStatusSuspended = "SUSPENDED"
	AdminStatusDeleted   = "DELETED"

	AdminAccessTokenExpiry  = 10 * time.Minute
	AdminRefreshTokenExpiry = 10 * time.Hour
)

var ValidRoles = []string{RoleSuperAdmin, RoleSupport, RoleReviewer, RoleEngineer}

func IsValidRole(role string) bool {
	for _, r := range ValidRoles {
		if r == role {
			return true
		}
	}
	return false
}
