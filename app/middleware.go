package app

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func JWTMiddlewareX(gc *gin.Context) {
	// Try to get token from cookie first
	tokenStr, err := gc.Cookie("access_token")
	if err != nil || tokenStr == "" {
		// Fallback: try to get token from Authorization header
		authHeader := gc.GetHeader("Authorization")
		if authHeader == "" {
			gc.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing Authorization token"})
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			gc.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid Authorization header format"})
			return
		}
		tokenStr = parts[1]
	}

	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return Singleton.JwtKey, nil
	})

	if err != nil || !token.Valid {
		gc.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
		return
	}

	// Token is valid, save user info in context
	gc.Set("claims", claims)
	gc.Set("user-id", claims.UserId)
	gc.Next()
}

// JWTMiddleware ...
func JWTMiddleware(gc *gin.Context) {
	var tokenStr string

	// 1. First try Authorization header
	authHeader := gc.GetHeader("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
	}

	// 2. If not in header, try cookie
	if tokenStr == "" {
		cookie, err := gc.Cookie("access_token")
		if err == nil {
			tokenStr = cookie
		}
	}

	if tokenStr == "" {
		gc.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
		return
	}

	// 3. Parse and validate
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return Singleton.JwtKey, nil
	})
	if err != nil || !token.Valid {
		gc.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	// 4. Store claims for downstream handlers
	gc.Set("user-id", claims.UserId)

	gc.Next()
}

// Get the id of the authorized user or -1
// userId will only be present if middleware sets it after verifying the JWT.
// If the user is not logged in or the middleware is not run, this will return -1.
func CurrentUserId(gc *gin.Context) (int, error) {
	userIdVal, exists := gc.Get("user-id")
	if exists {
		userId, ok := userIdVal.(int)
		if ok {
			return userId, nil
		}
	}
	return -1, fmt.Errorf("unauthorized user")
}
