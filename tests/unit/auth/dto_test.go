package auth_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"codeberg.org/azzet/azzetbe/internal/auth"
	"codeberg.org/azzet/azzetbe/internal/db"
	"codeberg.org/azzet/azzetbe/internal/shared"
)

func TestUserToResponse_WithEmail(t *testing.T) {
	email := "test@example.com"
	now := time.Now()

	user := &db.User{
		ID:               uuid.New(),
		Email:            pgtype.Text{String: email, Valid: true},
		Whatsapp:         pgtype.Text{Valid: false},
		EmailVerified:    true,
		WhatsappVerified: false,
		Status:           auth.StatusActive,
		CreatedAt:        now,
	}

	resp := auth.UserToResponse(user)

	if resp.ID != user.ID.String() {
		t.Fatalf("expected ID '%s', got '%s'", user.ID.String(), resp.ID)
	}
	if resp.Email == nil || *resp.Email != email {
		t.Fatalf("expected email '%s', got %v", email, resp.Email)
	}
	if resp.WhatsApp != nil {
		t.Fatalf("expected nil whatsapp, got %v", resp.WhatsApp)
	}
	if !resp.EmailVerified {
		t.Fatal("expected email_verified true")
	}
	if resp.WhatsAppVerified {
		t.Fatal("expected whatsapp_verified false")
	}
	if resp.Status != auth.StatusActive {
		t.Fatalf("expected status '%s', got '%s'", auth.StatusActive, resp.Status)
	}
}

func TestUserToResponse_WithWhatsApp(t *testing.T) {
	wa := "+628123456789"
	now := time.Now()

	user := &db.User{
		ID:               uuid.New(),
		Email:            pgtype.Text{Valid: false},
		Whatsapp:         pgtype.Text{String: wa, Valid: true},
		EmailVerified:    false,
		WhatsappVerified: true,
		Status:           auth.StatusActive,
		CreatedAt:        now,
	}

	resp := auth.UserToResponse(user)

	if resp.Email != nil {
		t.Fatalf("expected nil email, got %v", resp.Email)
	}
	if resp.WhatsApp == nil || *resp.WhatsApp != wa {
		t.Fatalf("expected whatsapp '%s', got %v", wa, resp.WhatsApp)
	}
	if !resp.WhatsAppVerified {
		t.Fatal("expected whatsapp_verified true")
	}
}

func TestSessionToResponse(t *testing.T) {
	now := time.Now()

	session := &db.Session{
		ID:         uuid.New(),
		UserID:     uuid.New(),
		DeviceName: pgtype.Text{String: "Chrome on MacOS", Valid: true},
		ExpiresAt:  now.Add(7 * 24 * time.Hour),
		LastUsedAt: now,
		CreatedAt:  now,
	}

	resp := auth.SessionToResponse(session)

	if resp.ID != session.ID.String() {
		t.Fatalf("expected ID '%s', got '%s'", session.ID.String(), resp.ID)
	}
	if resp.DeviceName == nil || *resp.DeviceName != "Chrome on MacOS" {
		t.Fatalf("expected device_name 'Chrome on MacOS', got %v", resp.DeviceName)
	}
	if resp.LastUsedAt == "" {
		t.Fatal("expected non-empty last_used_at")
	}
	if resp.CreatedAt == "" {
		t.Fatal("expected non-empty created_at")
	}
}

func TestSessionToResponse_NilFields(t *testing.T) {
	now := time.Now()

	session := &db.Session{
		ID:         uuid.New(),
		UserID:     uuid.New(),
		DeviceName: pgtype.Text{Valid: false},
		IpAddress:  nil,
		ExpiresAt:  now.Add(7 * 24 * time.Hour),
		LastUsedAt: now,
		CreatedAt:  now,
	}

	resp := auth.SessionToResponse(session)

	if resp.DeviceName != nil {
		t.Fatalf("expected nil device_name, got %v", resp.DeviceName)
	}
	if resp.IPAddress != nil {
		t.Fatalf("expected nil ip_address, got %v", resp.IPAddress)
	}
}

func TestConstants(t *testing.T) {
	if auth.StatusActive != "ACTIVE" {
		t.Fatalf("expected 'ACTIVE', got '%s'", auth.StatusActive)
	}
	if auth.StatusSuspended != "SUSPENDED" {
		t.Fatalf("expected 'SUSPENDED', got '%s'", auth.StatusSuspended)
	}
	if auth.StatusUnverified != "UNVERIFIED" {
		t.Fatalf("expected 'UNVERIFIED', got '%s'", auth.StatusUnverified)
	}
	if auth.StatusDeleted != "DELETED" {
		t.Fatalf("expected 'DELETED', got '%s'", auth.StatusDeleted)
	}
	if auth.IdentifierTypeEmail != "email" {
		t.Fatalf("expected 'email', got '%s'", auth.IdentifierTypeEmail)
	}
	if auth.IdentifierTypeWA != "whatsapp" {
		t.Fatalf("expected 'whatsapp', got '%s'", auth.IdentifierTypeWA)
	}
}

func TestHashToken_Deterministic(t *testing.T) {
	token := "some-refresh-token-value"
	hash1 := shared.HashOTP(token)
	hash2 := shared.HashOTP(token)

	if hash1 != hash2 {
		t.Fatal("expected deterministic hash")
	}
}

func TestHashToken_DifferentInputs(t *testing.T) {
	hash1 := shared.HashOTP("token-1")
	hash2 := shared.HashOTP("token-2")

	if hash1 == hash2 {
		t.Fatal("expected different hashes for different tokens")
	}
}
