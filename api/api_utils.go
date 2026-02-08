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

var debug = true

func debugf(format string, args ...any) {
	if debug {
		fmt.Printf("[DEBUG] "+format+"\n", args...)
	}
}

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

func (h *ApiHandler) userId(gc *gin.Context) int {
	return gc.GetInt("user-id")
}

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

// GetContextParamInt returns a pointer to an int if the parameter exists and is valid,
// or nil if it doesn't exist or can't be parsed as an integer.
func GetContextParamInt(gc *gin.Context, name string) (*int, bool) {
	val, exists := gc.GetQuery(name)
	if !exists {
		val = gc.PostForm(name)
		if val == "" {
			return nil, false
		}
	}

	i, err := strconv.Atoi(val)
	if err != nil {
		return nil, false
	}

	return &i, true
}

// GetContextParamIntDefault returns the int value of the parameter if it exists and is valid,
// otherwise it returns the provided default value.
func GetContextParamIntDefault(gc *gin.Context, name string, defaultVal int) int {
	val, exists := gc.GetQuery(name)
	if !exists {
		val = gc.PostForm(name)
		if val == "" {
			return defaultVal
		}
	}

	i, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}

	return i
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
		return app.UranusInstance.JwtKey, nil
	})

	if err != nil || !token.Valid {
		return -1
	}

	return claims.UserId
}

const dummyPasswordHash = "$2a$12$wGf6R8t2pFzq9yQmYv8y1u8y0v7E4Qv9ZJ8tQ6lH5E8QK3yQyZCwK"

// VerifyUserPassword reads password from request body, validates it against user Id.
// Returns true if password is valid, or writes JSON error response to context and returns false.
func (h *ApiHandler) VerifyUserPassword(gc *gin.Context, userId int) error {
	var body struct {
		Password string `json:"password"`
	}

	if err := gc.ShouldBindJSON(&body); err != nil {
		return fmt.Errorf("invalid request body")
	}

	if body.Password == "" {
		return fmt.Errorf("password is required")
	}

	var passwordHash string
	query := fmt.Sprintf(`SELECT password_hash FROM %s.user WHERE id = $1`, h.Config.DbSchema)
	err := h.DbPool.QueryRow(gc.Request.Context(), query, userId).Scan(&passwordHash)
	if err != nil {
		passwordHash = dummyPasswordHash
	}

	if app.ComparePasswords(passwordHash, body.Password) != nil {
		return fmt.Errorf("invalid password")
	}

	return nil
}

func IsEventReleaseStatus(fieldName string, value *string) (bool, error) {
	return ValidateEnum(fieldName, value,
		"draft", "review", "released", "cancelled", "deferred", "rescheduled",
	)
}

// ValidateEnum checks if val is one of allowed values. Returns an error message if invalid, else empty string.
func ValidateEnum(fieldName string, value *string, allowed ...string) (bool, error) {
	if value == nil {
		return false, fmt.Errorf("value must be != nil")
	}

	allowedSet := make(map[string]struct{}, len(allowed))
	for _, v := range allowed {
		allowedSet[v] = struct{}{}
	}

	if _, ok := allowedSet[*value]; !ok {
		return false, fmt.Errorf("%s must be one of %v", fieldName, allowed)
	}

	return true, nil
}
