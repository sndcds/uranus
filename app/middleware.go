package app

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func JWTMiddleware(gc *gin.Context) {
	var tokenStr string
	fromCookie := false

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
			fromCookie = true
		}
	}

	if tokenStr == "" {
		gc.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "missing token",
		})
		return
	}

	claims, err := ParseJWT(tokenStr)
	if err != nil || claims.TokenType != AccessTokenType || !ValidUUID(claims.UserUuid) {
		gc.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}
	if fromCookie && !SafeMethod(gc.Request.Method) && !RequireAuthOrigin(gc) {
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
