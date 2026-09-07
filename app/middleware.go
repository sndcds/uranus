package app

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func JWTMiddleware(gc *gin.Context) {
	var tokenStr string

	// 1. Try Authorization header first.
	authHeader := gc.GetHeader("Authorization")
	parts := strings.Fields(authHeader)

	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		tokenStr = parts[1]
	}

	// 2. If not in header, try access-token cookie.
	if tokenStr == "" {
		cookie, err := gc.Cookie("access_token")
		if err == nil {
			tokenStr = cookie
		}
	}

	if tokenStr == "" {
		gc.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "missing token",
		})
		return
	}

	// 3. Parse and validate token.
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(
		tokenStr,
		claims,
		func(token *jwt.Token) (any, error) {
			return UranusInstance.JwtKey, nil
		},
		jwt.WithValidMethods([]string{
			jwt.SigningMethodHS256.Alg(),
		}),
	)

	if err != nil || !token.Valid {
		gc.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "invalid token",
		})
		return
	}

	// 4. Only access tokens may authenticate API requests.
	if claims.TokenType != "access" {
		gc.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "invalid token",
		})
		return
	}

	if claims.UserUuid == "" {
		gc.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "invalid token",
		})
		return
	}

	// 5. Store authentication information for downstream handlers.
	gc.Set("user-uuid", claims.UserUuid)
	gc.Set("jwt-claims", claims)

	gc.Next()
}

func LocalhostOnlyMiddleware(gc *gin.Context) {
	fmt.Println("LocalhostOnlyMiddleware")

	ip := net.ParseIP(gc.ClientIP())

	if ip == nil || !ip.IsLoopback() {
		gc.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error": "localhost only",
		})
		return
	}

	gc.Next()
}
