package shared

import (
	"net/mail"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

type ValidationErrors map[string]string

func (ve ValidationErrors) Add(field, message string) {
	ve[field] = message
}

func (ve ValidationErrors) HasErrors() bool {
	return len(ve) > 0
}

func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("validation errors: ")
	first := true
	for field, msg := range ve {
		if !first {
			sb.WriteString("; ")
		}
		sb.WriteString(field)
		sb.WriteString(": ")
		sb.WriteString(msg)
		first = false
	}
	return sb.String()
}

func ValidateRequired(value, field, label string) string {
	if strings.TrimSpace(value) == "" {
		return label + " is required"
	}
	return ""
}

func ValidateEmail(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	_, err := mail.ParseAddress(value)
	if err != nil {
		return "invalid email format"
	}
	return ""
}

func ValidateMinLength(value string, min int, label string) string {
	if utf8.RuneCountInString(value) < min {
		return label + " must be at least " + strconv.Itoa(min) + " characters"
	}
	return ""
}

func ValidateMaxLength(value string, max int, label string) string {
	if utf8.RuneCountInString(value) > max {
		return label + " must not exceed " + strconv.Itoa(max) + " characters"
	}
	return ""
}

func ValidateRange(value, min, max int, label string) string {
	if value < min || value > max {
		return label + " must be between " + strconv.Itoa(min) + " and " + strconv.Itoa(max)
	}
	return ""
}

var uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func ValidateUUID(value string) string {
	if value == "" {
		return ""
	}
	if !uuidRegex.MatchString(strings.ToLower(value)) {
		return "invalid UUID format"
	}
	return ""
}

var phoneRegex = regexp.MustCompile(`^\+?[1-9]\d{6,14}$`)

func ValidatePhone(value string) string {
	if value == "" {
		return ""
	}
	if !phoneRegex.MatchString(value) {
		return "invalid phone number format"
	}
	return ""
}

func ValidateURL(value string) string {
	if value == "" {
		return ""
	}
	u, err := url.ParseRequestURI(value)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "invalid URL format"
	}
	return ""
}

func ValidateEnum(value string, allowed []string, label string) string {
	if value == "" {
		return ""
	}
	for _, a := range allowed {
		if value == a {
			return ""
		}
	}
	return label + " must be one of: " + strings.Join(allowed, ", ")
}


