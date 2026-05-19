package shared_test

import (
	"testing"

	"codeberg.org/azzet/azzetbe/internal/shared"
)

func TestHashPassword(t *testing.T) {
	password := "SecurePass123!"

	hash, err := shared.HashPassword(password)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}
	if hash == password {
		t.Fatal("hash should not equal plaintext password")
	}
}

func TestHashPassword_DifferentHashes(t *testing.T) {
	password := "SecurePass123!"

	hash1, _ := shared.HashPassword(password)
	hash2, _ := shared.HashPassword(password)

	// bcrypt generates different hashes for same input (due to salt)
	if hash1 == hash2 {
		t.Fatal("expected different hashes due to bcrypt salt")
	}
}

func TestVerifyPassword_Valid(t *testing.T) {
	password := "SecurePass123!"

	hash, _ := shared.HashPassword(password)

	if !shared.VerifyPassword(hash, password) {
		t.Fatal("expected password verification to pass")
	}
}

func TestVerifyPassword_Invalid(t *testing.T) {
	password := "SecurePass123!"

	hash, _ := shared.HashPassword(password)

	if shared.VerifyPassword(hash, "WrongPassword") {
		t.Fatal("expected password verification to fail for wrong password")
	}
}

func TestVerifyPassword_EmptyPassword(t *testing.T) {
	hash, _ := shared.HashPassword("SecurePass123!")

	if shared.VerifyPassword(hash, "") {
		t.Fatal("expected password verification to fail for empty password")
	}
}

func TestVerifyPassword_EmptyHash(t *testing.T) {
	if shared.VerifyPassword("", "password") {
		t.Fatal("expected password verification to fail for empty hash")
	}
}

func TestHashPassword_EmptyInput(t *testing.T) {
	hash, err := shared.HashPassword("")
	if err != nil {
		t.Fatalf("expected no error for empty password, got %v", err)
	}
	// bcrypt can hash empty strings
	if hash == "" {
		t.Fatal("expected non-empty hash even for empty password")
	}
}
