package shared

import (
	"crypto/rand"
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

func (s *OTPService) Generate() string {
	max := new(big.Int)
	max.Exp(big.NewInt(10), big.NewInt(int64(s.Length)), nil)

	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		// Fallback to UUID-based generation
		return fmt.Sprintf("%06d", uuid.New().ID()%1000000)
	}

	format := fmt.Sprintf("%%0%dd", s.Length)
	return fmt.Sprintf(format, n.Int64())
}

func GenerateUUID() string {
	return uuid.New().String()
}
