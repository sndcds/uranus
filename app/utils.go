package app

import (
	"github.com/golang-jwt/jwt/v5"
)

// TODO: Review code

// Claims struct for JWT
type Claims struct {
	UserId int `json:"user_id"`
	jwt.RegisteredClaims
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
