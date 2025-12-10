package app

import "fmt"

// TODO: Review code

// ToInt converts an interface{} to int safely.
// Returns the int value and true if successful, false otherwise.
func ToInt(value interface{}) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int8:
		return int(v), true
	case int16:
		return int(v), true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	case uint:
		return int(v), true
	case uint8:
		return int(v), true
	case uint16:
		return int(v), true
	case uint32:
		return int(v), true
	case uint64:
		return int(v), true
	case float32:
		return int(v), true
	case float64:
		return int(v), true
	default:
		fmt.Printf("ToInt: unexpected type %T for value %#v\n", value, value)
		return 0, false
	}
}

func ClampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func ClampFloat32(v, min, max float32) float32 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

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
