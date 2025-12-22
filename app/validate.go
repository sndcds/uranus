package app

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// ValidateOptionalNonEmptyString checks if an optional string pointer is non-empty.
// - If value is nil, it's considered valid.
// - If value is non-nil but empty or whitespace-only, it returns an error.
func ValidateOptionalNonEmptyString(fieldName string, value *string) error {
	if value != nil && strings.TrimSpace(*value) == "" {
		return fmt.Errorf("%s cannot be empty if provided", fieldName)
	}
	return nil
}

// ValidateOptionalUrl validates a pointer to a string as a URL.
// - If the pointer is nil, it's considered valid.
// - If the string is empty or whitespace, it's considered invalid.
// - Otherwise, it checks for valid URL format and http/https scheme.
func ValidateOptionalUrl(fieldName string, value *string) error {
	if value == nil {
		// No value provided → valid
		return nil
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		// Value is provided but empty → error
		return fmt.Errorf("%s cannot be empty if provided", fieldName)
	}

	// Validate URL format
	u, err := url.Parse(trimmed)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("%s must be a valid URL", fieldName)
	}

	// Ensure it starts with http:// or https://
	if !(strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://")) {
		return fmt.Errorf("%s must start with http:// or https://", fieldName)
	}

	return nil
}

// ValidateOptionalDate validates an optional date string in the format YYYY-MM-DD.
// - If the pointer is nil or empty, it is considered valid.
// - Otherwise, it checks if the value matches the format "2006-01-02".
func ValidateOptionalDate(fieldName string, value *string) error {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	if _, err := time.Parse("2006-01-02", trimmed); err != nil {
		return fmt.Errorf("%s must be in format YYYY-MM-DD", fieldName)
	}
	return nil
}

// ValidateOptionalTime validates an optional time string in the format HH:MM (24-hour).
// - If the pointer is nil or empty, it is considered valid.
// - Otherwise, it checks if the value matches the format "15:04".
func ValidateOptionalTime(fieldName string, value *string) error {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	if _, err := time.Parse("15:04", trimmed); err != nil {
		return fmt.Errorf("%s must be in format HH:MM (24-hour)", fieldName)
	}
	return nil
}
