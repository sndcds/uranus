package api

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/grains/grains_uuid"
	"github.com/sndcds/uranus/app"
)

// TODO: Review code

func debugf(format string, args ...any) {
	if app.UranusInstance.Config.DebugLevel == 1 {
		fmt.Printf("[Uranus API debug] "+format+"\n", args...)
	}
}

type EventDateRequest struct {
	EventUUID string
	DateUUID  string
	Lang      string
}

// Package-level variables
var (
	priceTypesOptionsQuery     string
	currenciesOptionsQuery     string
	eventOccasionsOptionsQuery string
	oncePriceTypes             sync.Once
	onceCurrencies             sync.Once
	onceEventOccasions         sync.Once
)

func (h *ApiHandler) userUuid(gc *gin.Context) string {
	return gc.GetString("user-uuid")
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
func ParamIntDefault(gc *gin.Context, name string, defaultValue int) int {
	paramStr := gc.Param(name)
	if paramStr == "" {
		return defaultValue
	}

	val, err := strconv.Atoi(paramStr)
	if err != nil {
		return defaultValue
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
	if val, exists := gc.GetQuery(name); exists {
		return val, true
	}
	if val, exists := gc.GetPostForm(name); exists {
		return val, true
	}
	return "", false
}

func GetContextParamWithDefault(gc *gin.Context, name string, defaultValue string) (string, bool) {
	if val, exists := GetContextParam(gc, name); exists {
		return val, true
	}

	return defaultValue, false
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
func GetContextParamIntDefault(gc *gin.Context, name string, defaultValue int) int {
	val, exists := gc.GetQuery(name)
	if !exists {
		val = gc.PostForm(name)
		if val == "" {
			return defaultValue
		}
	}

	i, err := strconv.Atoi(val)
	if err != nil {
		return defaultValue
	}

	return i
}

// Get the Authorization header
func UseUuidFromAccessToken(gc *gin.Context) string {
	authHeader := gc.GetHeader("Authorization")
	if authHeader == "" {
		return ""
	}

	// Expect header format: "Bearer <token>"
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}
	accessToken := parts[1]

	// Parse the token
	claims := &app.Claims{}
	token, err := jwt.ParseWithClaims(accessToken, claims, func(token *jwt.Token) (interface{}, error) {
		return app.UranusInstance.JwtKey, nil
	})

	if err != nil || !token.Valid {
		return ""
	}

	return claims.UserUuid
}

const dummyPasswordHash = "$2a$12$wGf6R8t2pFzq9yQmYv8y1u8y0v7E4Qv9ZJ8tQ6lH5E8QK3yQyZCwK"

// VerifyUserPassword reads password from request body, validates it against user.uuis.
// Returns true if password is valid, or writes JSON error response to context and returns false.
func (h *ApiHandler) VerifyUserPassword(gc *gin.Context, userUuid string) error {
	var body struct {
		Password string `json:"password"`
	}

	if err := gc.ShouldBindJSON(&body); err != nil {
		return errors.New("invalid request body")
	}

	if body.Password == "" {
		return errors.New("password is required")
	}

	var passwordHash string
	query := fmt.Sprintf(`SELECT password_hash FROM %s.user WHERE uuid = $1::uuid`, h.DbSchema)
	err := h.DbPool.QueryRow(gc.Request.Context(), query, userUuid).Scan(&passwordHash)
	if err != nil {
		err.Error()
	}

	if app.ComparePasswords(passwordHash, body.Password) != nil {
		return errors.New("invalid password")
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
		return false, errors.New("value must be != nil")
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

func (h *ApiHandler) ResolveEventDateUuidFromSlug(
	ctx context.Context,
	eventUuid string,
	slug string,
) (string, error) {

	// Slug format: YYYYMMDDHHMM
	if len(slug) != 12 {
		return "", fmt.Errorf("invalid date slug format")
	}
	startDate := slug[:8] // 20260602
	startTime := slug[8:] // 1200
	// optionally format:
	parsedDate := fmt.Sprintf("%s-%s-%s", startDate[0:4], startDate[4:6], startDate[6:8])
	parsedTime := fmt.Sprintf("%s:%s", startTime[0:2], startTime[2:4])
	var dateUuid string
	err := h.DbPool.QueryRow(ctx, app.UranusInstance.SqlResolveEventDateUuidFromSlug, eventUuid, parsedDate, parsedTime).
		Scan(&dateUuid)
	if err != nil {
		return "", err
	}
	return dateUuid, nil
}

func BuildDateSlug(startDate, startTime string) string {
	return startDate[:4] + startDate[5:7] + startDate[8:10] + startTime[:2] + startTime[3:5]
}

func (h *ApiHandler) ResolveEventDateRequest(gc *gin.Context, apiRequest *grains_api.Request) (*EventDateRequest, bool) {
	ctx := gc.Request.Context()

	eventUuid := gc.Param("eventUuid")
	if eventUuid == "" {
		apiRequest.Required("eventUuid is required")
		return nil, false
	}
	apiRequest.SetMeta("event_uuid", eventUuid)

	dateIdentifier := gc.Param("dateIdentifier")
	if dateIdentifier == "" {
		apiRequest.Required("dateIdentifier is required")
		return nil, false
	}
	apiRequest.SetMeta("date_identifier", dateIdentifier)

	var dateUuid string

	if grains_uuid.IsValidUuidv7(dateIdentifier) {
		dateUuid = dateIdentifier
	} else {
		resolvedDateUuid, err := h.ResolveEventDateUuidFromSlug(ctx, eventUuid, dateIdentifier)
		if err != nil {
			apiRequest.NotFound("event date not found")
			return nil, false
		}
		dateUuid = resolvedDateUuid
	}

	lang := gc.DefaultQuery("lang", "en")

	apiRequest.SetMeta("date_uuid", dateUuid)
	apiRequest.SetMeta("lang", lang)

	return &EventDateRequest{
		EventUUID: eventUuid,
		DateUUID:  dateUuid,
		Lang:      lang,
	}, true
}
