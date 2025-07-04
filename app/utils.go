package app

import (
	"fmt"
	"os"
	"strings"
)

// CombineFlags takes a slice of integers representing flag positions
// and combines them into a single uint64 bitmask.
//
// Each integer in the input slice should be in the range [0, 63], representing
// a bit position in the 64-bit unsigned integer. The function sets the bit at each
// of these positions to 1 in the result.
//
// For example, if flags = []int{0, 2, 5}, the result will have bits 0, 2, and 5 set,
// resulting in a value like: 0b00100101.
//
// Any flag values outside the range [0, 63] are ignored.
//
// Parameters:
//   - flags: A slice of integers representing positions of individual flags.
//
// Returns:
//   - A uint64 value with the corresponding bits set.
func CombineFlags(flags []int) uint64 {
	var result uint64 = 0
	for _, flag := range flags {
		if flag >= 0 && flag < 64 {
			result |= 1 << flag
		}
	}
	return result
}

func loadFileReplaceAllSchema(filePath string, replacement string, outString *string) error {
	return loadFileReplaceAllShortcuts(filePath, "{{schema}}", replacement, outString)
}

func loadFileReplaceAllShortcuts(filePath string, shortcut string, replacement string, outString *string) error {
	// Read the SQL file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Convert to string and replace {{schema}} with actual schema name
	s := string(content)
	s = strings.ReplaceAll(s, shortcut, replacement)
	*outString = s
	return nil
}
