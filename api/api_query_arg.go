package api

import (
	"encoding/json"
	"fmt"
)

// NullableField wraps a value that can be null or absent in JSON.
type NullableField[T any] struct {
	Set   bool // true if field was present in JSON
	Value *T   // actual value, nil if JSON null
}

// UnmarshalJSON implements custom JSON binding
func (n *NullableField[T]) UnmarshalJSON(b []byte) error {
	n.Set = true
	if string(b) == "null" {
		n.Value = nil
		return nil
	}
	var val T
	if err := json.Unmarshal(b, &val); err != nil {
		return fmt.Errorf("invalid value: %w", err)
	}
	n.Value = &val
	return nil
}

// addUpdateClauseNullable adds a SQL SET clause if the field was present in JSON.
// Handles null values correctly.
func addUpdateClauseNullable[T any](fieldName string, value NullableField[T], setClauses *[]string, args *[]interface{}, argPos int) int {
	if value.Set {
		*setClauses = append(*setClauses, fmt.Sprintf("%s = $%d", fieldName, argPos))
		*args = append(*args, value.Value) // nil if JSON null
		argPos++
	}
	return argPos
}

func addStringUpdateClause(fieldName string, valuePtr *string, setClauses *[]string, args *[]interface{}, argPos int) int {
	if valuePtr != nil {
		*setClauses = append(*setClauses, fmt.Sprintf("%s = $%d", fieldName, argPos))
		if *valuePtr == "" {
			*args = append(*args, nil) // treat empty string as NULL
		} else {
			*args = append(*args, *valuePtr)
		}
		argPos++
	}
	return argPos
}

func addUpdateStringSliceField(
	fieldName string,
	valuePtr *[]string,
	setClauses *[]string,
	args *[]interface{},
	argPos int,
) int {
	if valuePtr != nil {
		*setClauses = append(*setClauses, fmt.Sprintf("%s = $%d", fieldName, argPos))
		if *valuePtr == nil || len(*valuePtr) == 0 {
			// NOT NULL array column → use empty array
			*args = append(*args, []string{})
		} else {
			*args = append(*args, *valuePtr)
		}
		argPos++
	}
	return argPos
}
