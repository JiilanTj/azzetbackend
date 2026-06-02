package storage

import (
	"path/filepath"
	"strings"
)

// SanitizeFilename strips path components and limits characters for safe object keys.
func SanitizeFilename(filename string) string {
	filename = filepath.Base(strings.TrimSpace(filename))
	var b strings.Builder
	for _, r := range filename {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	sanitized := b.String()
	if sanitized == "" || sanitized == "." {
		return "file"
	}
	return sanitized
}
