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

func (nf *NullableField[T]) UnmarshalJSON(data []byte) error {
	nf.Set = true // field is present
	if string(data) == "null" {
		nf.Value = nil
		return nil
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	nf.Value = &v
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

func addUpdateClauseString(fieldName string, valuePtr *string, setClauses *[]string, args *[]interface{}, argPos int) int {
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

func addUpdateClauseStringSliceField(
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
