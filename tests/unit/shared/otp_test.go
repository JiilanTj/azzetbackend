package shared_test

import (
	"testing"

	"codeberg.org/azzet/azzetbe/internal/shared"
)

func TestOTPGenerate_Length(t *testing.T) {
	svc := shared.NewOTPService(6)

	code := svc.Generate()
	if len(code) != 6 {
		t.Fatalf("expected OTP length 6, got %d: '%s'", len(code), code)
	}
}

func TestOTPGenerate_NumericOnly(t *testing.T) {
	svc := shared.NewOTPService(6)

	for i := 0; i < 100; i++ {
		code := svc.Generate()
		for _, c := range code {
			if c < '0' || c > '9' {
				t.Fatalf("expected numeric OTP, got '%s'", code)
			}
		}
	}
}

func TestOTPGenerate_Unique(t *testing.T) {
	svc := shared.NewOTPService(6)

	codes := make(map[string]bool)
	for i := 0; i < 100; i++ {
		code := svc.Generate()
		codes[code] = true
	}

	// With 6-digit codes and 100 generations, we should have mostly unique codes
	// Allow some collisions but not all the same
	if len(codes) < 50 {
		t.Fatalf("expected mostly unique OTPs, got only %d unique out of 100", len(codes))
	}
}

func TestOTPGenerate_CustomLength(t *testing.T) {
	svc := shared.NewOTPService(4)

	code := svc.Generate()
	if len(code) != 4 {
		t.Fatalf("expected OTP length 4, got %d: '%s'", len(code), code)
	}
}

func TestOTPGenerate_DefaultLength(t *testing.T) {
	svc := shared.NewOTPService(0) // Should default to 6

	code := svc.Generate()
	if len(code) != 6 {
		t.Fatalf("expected default OTP length 6, got %d: '%s'", len(code), code)
	}
}

func TestHashOTP(t *testing.T) {
	code := "123456"
	hash := shared.HashOTP(code)

	if hash == "" {
		t.Fatal("expected non-empty hash")
	}
	if hash == code {
		t.Fatal("hash should not equal plaintext")
	}
	// SHA256 hex is always 64 chars
	if len(hash) != 64 {
		t.Fatalf("expected SHA256 hash length 64, got %d", len(hash))
	}
}

func TestHashOTP_Deterministic(t *testing.T) {
	code := "654321"
	hash1 := shared.HashOTP(code)
	hash2 := shared.HashOTP(code)

	if hash1 != hash2 {
		t.Fatal("expected same hash for same input")
	}
}

func TestHashOTP_DifferentInputs(t *testing.T) {
	hash1 := shared.HashOTP("123456")
	hash2 := shared.HashOTP("654321")

	if hash1 == hash2 {
		t.Fatal("expected different hashes for different inputs")
	}
}

func TestVerifyOTP_Valid(t *testing.T) {
	code := "123456"
	hash := shared.HashOTP(code)

	if !shared.VerifyOTP(code, hash) {
		t.Fatal("expected OTP verification to pass")
	}
}

func TestVerifyOTP_Invalid(t *testing.T) {
	hash := shared.HashOTP("123456")

	if shared.VerifyOTP("654321", hash) {
		t.Fatal("expected OTP verification to fail for wrong code")
	}
}

func TestVerifyOTP_EmptyCode(t *testing.T) {
	hash := shared.HashOTP("123456")

	if shared.VerifyOTP("", hash) {
		t.Fatal("expected OTP verification to fail for empty code")
	}
}

func TestGenerateUUID(t *testing.T) {
	id1 := shared.GenerateUUID()
	id2 := shared.GenerateUUID()

	if id1 == "" {
		t.Fatal("expected non-empty UUID")
	}
	if id1 == id2 {
		t.Fatal("expected unique UUIDs")
	}
	// UUID format: 8-4-4-4-12
	if len(id1) != 36 {
		t.Fatalf("expected UUID length 36, got %d", len(id1))
	}
}
