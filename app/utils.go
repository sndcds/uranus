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
