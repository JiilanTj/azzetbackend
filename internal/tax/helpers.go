package tax

import (
	"fmt"
	"math/big"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func floatToNumeric(f float64) pgtype.Numeric {
	cents := int64(f * 100)
	return pgtype.Numeric{
		Int:   big.NewInt(cents),
		Exp:   -2,
		Valid: true,
	}
}

func numericToFloat(n pgtype.Numeric) float64 {
	if !n.Valid || n.Int == nil {
		return 0
	}
	f := new(big.Float).SetInt(n.Int)
	exp := n.Exp
	if exp == 0 {
		result, _ := f.Float64()
		return result
	}
	scale := new(big.Float).SetFloat64(1)
	for i := int32(0); i < absInt32(exp); i++ {
		if exp < 0 {
			scale.Mul(scale, big.NewFloat(10))
		} else {
			scale.Quo(scale, big.NewFloat(10))
		}
	}
	f.Mul(f, scale)
	result, _ := f.Float64()
	return result
}

func absInt32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}

func uuidToPgtype(id uuid.UUID) pgtype.UUID {
	if id == uuid.Nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: id, Valid: true}
}

func pgtypeUUIDToString(id pgtype.UUID) string {
	if !id.Valid {
		return ""
	}
	return uuid.UUID(id.Bytes).String()
}

func stringToPgtext(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

func pgtextToString(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}

func numericToString(n pgtype.Numeric) string {
	return fmt.Sprintf("%.2f", numericToFloat(n))
}

func interfaceToFloat(v interface{}) float64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int64:
		return float64(val)
	case pgtype.Numeric:
		return numericToFloat(val)
	default:
		var n pgtype.Numeric
		if err := n.Scan(v); err == nil {
			return numericToFloat(n)
		}
		return 0
	}
}
