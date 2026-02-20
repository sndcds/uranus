package api

import (
	"errors"
	"fmt"
	"time"
)

// ParseSqlDate validates and parses a date string in YYYY-MM-DD format.
// Returns a pointer to time.Time if valid, or an error if invalid.
// Returns nil if the input is empty.
func ParseSqlDateString(dateStr string) (*time.Time, error) {
	const layout = "2006-01-02"

	trimmed := dateStr
	if trimmed == "" {
		return nil, nil
	}

	t, err := time.Parse(layout, trimmed)
	if err != nil {
		return nil, errors.New("invalid date format, expected YYYY-MM-DD")
	}

	return &t, nil
}

// ParseSqlTime validates and parses a time string in HH:MM or HH:MM:SS format.
// Returns a pointer to time.Time (date part is zeroed) or an error if invalid.
// Returns nil if the input is empty.
func ParseSqlTimeString(timeStr string) (*time.Time, error) {
	if timeStr == "" {
		return nil, nil
	}

	// Try HH:MM:SS first
	layouts := []string{
		"15:04:05",
		"15:04", // fallback HH:MM
	}

	var t time.Time
	var err error
	for _, layout := range layouts {
		t, err = time.Parse(layout, timeStr)
		if err == nil {
			return &t, nil
		}
	}

	return nil, errors.New("invalid time format, expected HH:MM or HH:MM:SS")
}

func ParseNullableDateString(field NullableField[string], fieldName string, dateLayout string) (*time.Time, bool, error) {
	if !field.Set {
		// Not sent at all
		return nil, false, nil
	}
	if field.Value == nil {
		// Explicitly null
		return nil, true, nil
	}
	// Parse provided string
	t, err := time.Parse(dateLayout, *field.Value)
	if err != nil {
		return nil, true, fmt.Errorf("invalid %s format (expected %s)", fieldName, dateLayout)
	}
	return &t, true, nil
}

func ParseNullableTimeString(field NullableField[string], fieldName string, timeLayout string) (*time.Time, bool, error) {
	if !field.Set {
		// Not sent at all
		return nil, false, nil
	}
	if field.Value == nil {
		// Explicitly null
		return nil, true, nil
	}
	// Parse provided string
	t, err := time.Parse(timeLayout, *field.Value)
	if err != nil {
		return nil, true, fmt.Errorf("invalid %s format (expected %s)", fieldName, timeLayout)
	}
	return &t, true, nil
}
