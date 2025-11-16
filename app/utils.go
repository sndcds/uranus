package app

import (
	"fmt"
	"net/mail"

	"github.com/golang-jwt/jwt/v5"
)

// Claims struct for JWT
type Claims struct {
	UserId int `json:"user_id"`
	jwt.RegisteredClaims
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

// GenerateWKT takes lat/lon strings and returns a WKT POINT string
func GenerateWKT(lat, lon float64) (string, error) {
	wkt := fmt.Sprintf("POINT(%f %f)", lon, lat)
	return wkt, nil
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

func IsValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

// FilterNilMap removes nil values from a single map
func FilterNilMap[T ~map[string]interface{}](data T) T {
	filtered := make(T)
	for k, v := range data {
		if v != nil {
			filtered[k] = v
		}
	}
	return filtered
}

// FilterNilSlice removes nil values from each map in a slice
func FilterNilSlice[T ~map[string]interface{}](data []T) []T {
	filteredSlice := make([]T, 0, len(data))
	for _, item := range data {
		filteredSlice = append(filteredSlice, FilterNilMap(item))
	}
	return filteredSlice
}
