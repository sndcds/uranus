package app

import (
	"errors"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	AccessTokenType  = "access"
	RefreshTokenType = "refresh"
)

// ParseJWT validates the signature and registered claims, not the token's purpose.
// Exported because authentication handlers live in package api.
func ParseJWT(tokenString string) (*Claims, error) {
	if UranusInstance == nil || len(UranusInstance.JwtKey) == 0 {
		return nil, errors.New("JWT signing key is not configured")
	}
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(_ *jwt.Token) (any, error) {
		return UranusInstance.JwtKey, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// ValidUUID accepts existing UUID versions, but rejects empty and nil identities.
func ValidUUID(value string) bool {
	id, err := uuid.Parse(value)
	return err == nil && id != uuid.Nil && len(value) == 36
}
