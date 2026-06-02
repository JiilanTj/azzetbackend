package shared

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/google/uuid"
)

type OTPService struct {
	Length int
}

func NewOTPService(length int) *OTPService {
	if length <= 0 {
		length = 6
	}
	return &OTPService{Length: length}
}

// Generate creates a cryptographically random OTP code
func (s *OTPService) Generate() string {
	max := new(big.Int)
	max.Exp(big.NewInt(10), big.NewInt(int64(s.Length)), nil)

	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		b := make([]byte, 8)
		if _, rerr := rand.Read(b); rerr != nil {
			panic("crypto/rand unavailable: " + rerr.Error())
		}
		n = new(big.Int).SetBytes(b)
		n.Mod(n, max)
	}

	format := fmt.Sprintf("%%0%dd", s.Length)
	return fmt.Sprintf(format, n.Int64())
}

// HashOTP hashes an OTP code with SHA256 for secure storage
func HashOTP(code string) string {
	h := sha256.Sum256([]byte(code))
	return hex.EncodeToString(h[:])
}

// VerifyOTP compares a plaintext OTP against a stored hash
func VerifyOTP(code, hash string) bool {
	return subtle.ConstantTimeCompare([]byte(HashOTP(code)), []byte(hash)) == 1
}

func GenerateUUID() string {
	return uuid.New().String()
}
