package shared_test

import (
	"testing"
	"time"

	"codeberg.org/azzet/azzetbe/internal/shared"
)

func TestGenerateAccessToken(t *testing.T) {
	svc := shared.NewJWTService("test-access-secret", "test-refresh-secret", 15*time.Minute, 7*24*time.Hour)

	token, err := svc.GenerateAccessToken("user-123", "session-456")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
}

func TestGenerateRefreshToken(t *testing.T) {
	svc := shared.NewJWTService("test-access-secret", "test-refresh-secret", 15*time.Minute, 7*24*time.Hour)

	token, err := svc.GenerateRefreshToken("user-123", "session-456")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
}

func TestValidateAccessToken_Valid(t *testing.T) {
	svc := shared.NewJWTService("test-access-secret", "test-refresh-secret", 15*time.Minute, 7*24*time.Hour)

	token, _ := svc.GenerateAccessToken("user-123", "session-456")

	claims, err := svc.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if claims.UserID != "user-123" {
		t.Fatalf("expected user_id 'user-123', got '%s'", claims.UserID)
	}
	if claims.SessionID != "session-456" {
		t.Fatalf("expected session_id 'session-456', got '%s'", claims.SessionID)
	}
	if claims.JTI == "" {
		t.Fatal("expected non-empty JTI")
	}
}

func TestValidateAccessToken_InvalidSecret(t *testing.T) {
	svc1 := shared.NewJWTService("secret-1", "refresh-1", 15*time.Minute, 7*24*time.Hour)
	svc2 := shared.NewJWTService("secret-2", "refresh-2", 15*time.Minute, 7*24*time.Hour)

	token, _ := svc1.GenerateAccessToken("user-123", "session-456")

	_, err := svc2.ValidateAccessToken(token)
	if err == nil {
		t.Fatal("expected error for invalid secret")
	}
}

func TestValidateRefreshToken_Valid(t *testing.T) {
	svc := shared.NewJWTService("test-access-secret", "test-refresh-secret", 15*time.Minute, 7*24*time.Hour)

	token, _ := svc.GenerateRefreshToken("user-123", "session-456")

	claims, err := svc.ValidateRefreshToken(token)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if claims.UserID != "user-123" {
		t.Fatalf("expected user_id 'user-123', got '%s'", claims.UserID)
	}
	if claims.SessionID != "session-456" {
		t.Fatalf("expected session_id 'session-456', got '%s'", claims.SessionID)
	}
}

func TestValidateRefreshToken_CannotUseAccessSecret(t *testing.T) {
	svc := shared.NewJWTService("test-access-secret", "test-refresh-secret", 15*time.Minute, 7*24*time.Hour)

	// Generate access token
	accessToken, _ := svc.GenerateAccessToken("user-123", "session-456")

	// Try to validate as refresh token (should fail - different secret)
	_, err := svc.ValidateRefreshToken(accessToken)
	if err == nil {
		t.Fatal("expected error when validating access token as refresh token")
	}
}

func TestValidateAccessToken_Expired(t *testing.T) {
	// Create service with 0 expiry (immediately expired)
	svc := shared.NewJWTService("test-access-secret", "test-refresh-secret", -1*time.Second, 7*24*time.Hour)

	token, _ := svc.GenerateAccessToken("user-123", "session-456")

	_, err := svc.ValidateAccessToken(token)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestValidateAccessToken_Malformed(t *testing.T) {
	svc := shared.NewJWTService("test-access-secret", "test-refresh-secret", 15*time.Minute, 7*24*time.Hour)

	_, err := svc.ValidateAccessToken("not.a.valid.token")
	if err == nil {
		t.Fatal("expected error for malformed token")
	}
}

func TestValidateAccessToken_EmptyString(t *testing.T) {
	svc := shared.NewJWTService("test-access-secret", "test-refresh-secret", 15*time.Minute, 7*24*time.Hour)

	_, err := svc.ValidateAccessToken("")
	if err == nil {
		t.Fatal("expected error for empty token")
	}
}

func TestTokensAreUnique(t *testing.T) {
	svc := shared.NewJWTService("test-access-secret", "test-refresh-secret", 15*time.Minute, 7*24*time.Hour)

	token1, _ := svc.GenerateAccessToken("user-123", "session-1")
	token2, _ := svc.GenerateAccessToken("user-123", "session-2")

	if token1 == token2 {
		t.Fatal("expected unique tokens for different sessions")
	}
}

func TestJTIIsUnique(t *testing.T) {
	svc := shared.NewJWTService("test-access-secret", "test-refresh-secret", 15*time.Minute, 7*24*time.Hour)

	token1, _ := svc.GenerateAccessToken("user-123", "session-1")
	token2, _ := svc.GenerateAccessToken("user-123", "session-1")

	claims1, _ := svc.ValidateAccessToken(token1)
	claims2, _ := svc.ValidateAccessToken(token2)

	if claims1.JTI == claims2.JTI {
		t.Fatal("expected unique JTI for each token")
	}
}
