package accounting

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// --- pgtype conversion helpers ---

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
	case int32:
		return float64(val)
	case int:
		return float64(val)
	case pgtype.Numeric:
		return numericToFloat(val)
	default:
		// Try to scan as numeric
		var n pgtype.Numeric
		if err := n.Scan(v); err == nil {
			return numericToFloat(n)
		}
		return 0
	}
}

func interfaceToString(v interface{}) string {
	if v == nil {
		return "0.00"
	}
	f := interfaceToFloat(v)
	return fmt.Sprintf("%.2f", f)
}
