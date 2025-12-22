package api

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sndcds/uranus/app"
)

// TODO: Review code

// Package-level variables
var (
	priceTypesOptionsQuery     string
	currenciesOptionsQuery     string
	eventOccasionsOptionsQuery string
	oncePriceTypes             sync.Once
	onceCurrencies             sync.Once
	onceEventOccasions         sync.Once
)

// ParamInt extracts a URL path parameter as an integer.
// If conversion fails, returns (0, false).
func ParamInt(gc *gin.Context, name string) (int, bool) {
	paramStr := gc.Param(name)
	val, err := strconv.Atoi(paramStr)
	if err != nil {
		return 0, false
	}
	return val, true
}

// ParamIntDefault extracts a URL path parameter as an integer.
// If conversion fails or the parameter is missing, returns the provided default value.
func ParamIntDefault(gc *gin.Context, name string, defaultVal int) int {
	paramStr := gc.Param(name)
	if paramStr == "" {
		return defaultVal
	}

	val, err := strconv.Atoi(paramStr)
	if err != nil {
		return defaultVal
	}

	return val
}

// getPostFormPtr returns a *string pointing to the form value if present, or nil if not.
func getPostFormPtr(gc *gin.Context, field string) *string {
	if val, ok := gc.GetPostForm(field); ok {
		return &val
	}
	return nil
}

// getPostFormIntPtr returns a *int from a form field if present and valid.
// Returns nil if field is missing, empty, or zero.
func getPostFormIntPtr(gc *gin.Context, field string) (*int, error) {
	valStr := gc.PostForm(field)
	if valStr == "" {
		return nil, nil
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return nil, fmt.Errorf("invalid %s: %v", field, err)
	}
	if val == 0 {
		return nil, nil // treat 0 as "not set"
	}
	return &val, nil
}

func getPostFormFloatPtr(gc *gin.Context, key string) (*float64, error) {
	val := strings.TrimSpace(gc.PostForm(key))
	if val == "" {
		return nil, nil
	}
	// allow both "," and "." as decimal separator
	val = strings.ReplaceAll(val, ",", ".")
	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid %s: %q", key, gc.PostForm(key))
	}
	return &f, nil
}

// GetContextParam attempts to retrieve a parameter value from the Gin context.
//
// It first checks for a query parameter with the given name (e.g., /endpoint?name=value).
// If the query parameter is not present, it then checks the POST form body for the same parameter.
// If neither is found, or if the form parameter is an empty string, it returns false.
//
// Parameters:
//   - gc: the *gin.Context containing the HTTP request context.
//   - name: the name of the parameter to retrieve.
//
// Returns:
//   - string: the value of the parameter, if found.
//   - bool: true if the parameter was found in either query or form data and is non-empty; false otherwise.
func GetContextParam(gc *gin.Context, name string) (string, bool) {
	val, exists := gc.GetQuery(name)
	if exists {

		return val, true
	}
	val = gc.PostForm(name)
	if val != "" {
		return val, true
	}
	return "", false
}

// Get the Authorization header
func UserIdFromAccessToken(gc *gin.Context) int {
	authHeader := gc.GetHeader("Authorization")
	if authHeader == "" {
		return -1
	}

	// Expect header format: "Bearer <token>"
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return -1
	}
	accessToken := parts[1]

	// Parse the token
	claims := &app.Claims{}
	token, err := jwt.ParseWithClaims(accessToken, claims, func(token *jwt.Token) (interface{}, error) {
		return app.Singleton.JwtKey, nil
	})

	if err != nil || !token.Valid {
		return -1
	}

	return claims.UserId
}
