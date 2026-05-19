package admin

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

const (
	mfaIssuer = "Azzet Admin"
	mfaDigits = 6
	mfaPeriod = 30
)

// GenerateMFASecret creates a new TOTP secret for an admin
func GenerateMFASecret(email string) (secret string, qrURL string, err error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      mfaIssuer,
		AccountName: email,
		Period:      mfaPeriod,
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA1,
		SecretSize:  20,
		Rand:        rand.Reader,
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to generate TOTP key: %w", err)
	}

	return key.Secret(), key.URL(), nil
}

// ValidateMFACode verifies a TOTP code against the secret
func ValidateMFACode(secret, code string) bool {
	return totp.Validate(code, secret)
}

// GenerateRandomSecret generates a random base32 secret (fallback)
func GenerateRandomSecret(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(bytes), nil
}
