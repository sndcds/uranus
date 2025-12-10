package app

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// TODO: Review code

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
	if claims.UserId < 0 {
		gc.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid user ID"})
	}

	// 4. Store claims for downstream handlers
	gc.Set("user-id", claims.UserId)

	gc.Next()
}
