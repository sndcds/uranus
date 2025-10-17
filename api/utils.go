package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sndcds/uranus/app"
)

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

func ParamAsIntMessageOnFail(gc *gin.Context, param string) (int, bool) {
	valueStr := gc.Param(param)
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{
			"message": fmt.Sprintf("Uranus server error: 400 (Bad Request) %s .. %s is not a integer number",
				gc.FullPath(),
				param),
		})
		return 0, false
	}
	return value, true
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
