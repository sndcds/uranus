package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sndcds/uranus/app"
)

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
//
// Example usage:
//
//	if val, ok := GetContextParam(c, "user_id"); ok {
//	    // use val
//	} else {
//	    // handle required parameter
//	}
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

func GetContextParameterAsInt(c *gin.Context, name string) (int, bool) {
	valStr, ok := GetContextParam(c, name)
	if !ok {
		return 0, false
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return 0, false
	}
	return val, true
}

func GetOptionalIntQueryParam(c *gin.Context, param string, defaultVal int) (int, error) {
	valStr := c.Query(param)
	if valStr == "" {
		return defaultVal, nil
	}
	return strconv.Atoi(valStr)
}

// StringToIntWithDefault converts a pointer to string 's' to an integer.
// If 's' is nil or points to an empty string, it returns the provided default value and true.
// If the conversion fails, it returns the default value and false.
func StringToIntWithDefault(s *string, defaultValue int) (int, bool) {
	if s == nil || *s == "" {
		return defaultValue, true
	}
	val, err := strconv.Atoi(*s)
	if err != nil {
		return defaultValue, false
	}
	return val, true
}

func InternalServerErrorAnswer(gc *gin.Context, err error) bool {
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{
			"message": fmt.Sprintf("Uranus server error: 500 (InternalServerError) %s .. %s",
				gc.FullPath(),
				err.Error()),
		})
		return true
	}
	return false
}

func UserIdFromAccessToken(gc *gin.Context) int {
	// Get the Authorization header
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

// GetUserPermissionBits returns the bitwise AND of user permissions and the given bit mask.
func (h *ApiHandler) GetUserPermissionBits(
	gc *gin.Context, tx pgx.Tx, userID, organizerID int, bitMask int64,
) (int64, error) {
	ctx := gc.Request.Context()
	var result pgtype.Int8
	err := tx.QueryRow(ctx,
		app.Singleton.SqlGetUserOrganizerPermissions,
		userID, organizerID, bitMask,
	).Scan(&result)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	if !result.Valid {
		return 0, nil
	}
	return result.Int64, nil
}
