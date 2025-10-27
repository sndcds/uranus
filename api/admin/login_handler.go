package api_admin

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

func LoginHandler(gc *gin.Context) {

	fmt.Println("....1")
	var credentials struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	fmt.Println("....2")
	if err := gc.BindJSON(&credentials); err != nil || credentials.Email == "" || credentials.Password == "" {
		gc.JSON(http.StatusUnauthorized, gin.H{"error": "credentials required"})
		return
	}

	fmt.Println("....3")
	user, err := model.GetUser(app.Singleton, credentials.Email)
	fmt.Println("....3 err: ", err)
	fmt.Println("....3 compare: ", app.ComparePasswords(user.PasswordHash, credentials.Password))

	if err != nil || app.ComparePasswords(user.PasswordHash, credentials.Password) != nil {
		gc.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	fmt.Println("....4")
	// -----------------------
	// Create tokens
	// -----------------------
	accessExp := time.Now().Add(time.Duration(app.Singleton.Config.AuthTokenExpirationTime) * time.Second)
	refreshExp := time.Now().Add(7 * 24 * time.Hour)

	accessClaims := &app.Claims{
		UserId: user.Id,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExp),
		},
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenStr, err := accessToken.SignedString(app.Singleton.JwtKey)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create access token"})
		return
	}

	fmt.Println("....5")
	refreshClaims := &app.Claims{
		UserId: user.Id,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshExp),
		},
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenStr, err := refreshToken.SignedString(app.Singleton.JwtKey)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create refresh token"})
		return
	}

	fmt.Println("....6")
	gc.JSON(http.StatusOK, gin.H{
		"message":       "login successful",
		"user_id":       user.Id,
		"display_name":  user.DisplayName,
		"locale":        user.Locale,
		"theme":         user.Theme,
		"access_token":  accessTokenStr,
		"refresh_token": refreshTokenStr,
	})
}

func RefreshHandler(gc *gin.Context) {
	// Get refresh token from Authorization header
	authHeader := gc.GetHeader("Authorization")
	if authHeader == "" {
		gc.JSON(http.StatusUnauthorized, gin.H{"error": "authorization header missing"})
		return
	}

	// Expected format: "Bearer <token>"
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		gc.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
		return
	}
	refreshToken := parts[1]

	// Parse and validate token
	claims := &app.Claims{}
	tkn, err := jwt.ParseWithClaims(refreshToken, claims, func(token *jwt.Token) (interface{}, error) {
		return app.Singleton.JwtKey, nil
	})
	if err != nil || !tkn.Valid {
		gc.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}

	// Issue new access token
	accessExp := time.Now().Add(time.Duration(app.Singleton.Config.AuthTokenExpirationTime) * time.Second)
	newClaims := &app.Claims{
		UserId: claims.UserId,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExp),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, newClaims)
	accessTokenStr, err := accessToken.SignedString(app.Singleton.JwtKey)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to sign access token"})
		return
	}

	// Send new access token back in header (and optionally JSON body)
	gc.Header("Authorization", "Bearer "+accessTokenStr)
	gc.JSON(http.StatusOK, gin.H{
		"message":      "token refreshed",
		"access_token": accessTokenStr,
		"expires_in":   int(time.Until(accessExp).Seconds()),
	})
}
