package api

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

func (h *ApiHandler) Login(gc *gin.Context) {
	var credentials struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	const loginErrorMsg = "invalid email or password"

	// Parse credentials
	if err := gc.BindJSON(&credentials); err != nil || credentials.Email == "" || credentials.Password == "" {
		gc.JSON(http.StatusUnauthorized, gin.H{"error": loginErrorMsg})
		return
	}

	// Load user
	user, err := model.GetUser(app.Singleton, credentials.Email)
	if err != nil || !user.IsActive || app.ComparePasswords(user.PasswordHash, credentials.Password) != nil {
		log.Printf("Login failed for email=%s: err=%v, active=%v", credentials.Email, err, user.IsActive)
		gc.JSON(http.StatusUnauthorized, gin.H{"error": loginErrorMsg})
		return
	}

	// Create access token
	accessExp := time.Now().Add(time.Duration(h.Config.AuthTokenExpirationTime) * time.Second)
	accessClaims := &app.Claims{
		UserId: user.Id,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExp),
		},
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenStr, err := accessToken.SignedString(app.Singleton.JwtKey)
	if err != nil {
		log.Printf("Failed to sign access token for user=%d: %v", user.Id, err)
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	// Create refresh token
	refreshExp := time.Now().Add(7 * 24 * time.Hour)
	refreshClaims := &app.Claims{
		UserId: user.Id,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshExp),
		},
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenStr, err := refreshToken.SignedString(app.Singleton.JwtKey)
	if err != nil {
		log.Printf("Failed to sign refresh token for user=%d: %v", user.Id, err)
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	// Return tokens
	gc.JSON(http.StatusOK, gin.H{
		"message":       "login successful",
		"user_id":       user.Id,
		"display_name":  user.DisplayName,
		"first_name":    user.FirstName,
		"last_name":     user.LastName,
		"locale":        user.Locale,
		"theme":         user.Theme,
		"access_token":  accessTokenStr,
		"refresh_token": refreshTokenStr,
	})
}

func (h *ApiHandler) Refresh(gc *gin.Context) {
	const refreshErrorMsg = "invalid refresh token"

	// Get token from Authorization header
	authHeader := gc.GetHeader("Authorization")
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		log.Printf("Invalid refresh header: %s", authHeader)
		gc.JSON(http.StatusUnauthorized, gin.H{"error": refreshErrorMsg})
		return
	}
	refreshToken := parts[1]

	// Parse token
	claims := &app.Claims{}
	tkn, err := jwt.ParseWithClaims(refreshToken, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return app.Singleton.JwtKey, nil
	})
	if err != nil || !tkn.Valid {
		log.Printf("Invalid refresh token: %v", err)
		gc.JSON(http.StatusUnauthorized, gin.H{"error": refreshErrorMsg})
		return
	}

	// Issue new access token
	accessExp := time.Now().Add(time.Duration(h.Config.AuthTokenExpirationTime) * time.Second)
	newClaims := &app.Claims{
		UserId: claims.UserId,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExp),
		},
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, newClaims)
	accessTokenStr, err := accessToken.SignedString(app.Singleton.JwtKey)
	if err != nil {
		log.Printf("Failed to sign new access token for user=%d: %v", claims.UserId, err)
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	// Return new access token
	gc.Header("Authorization", "Bearer "+accessTokenStr)
	gc.JSON(http.StatusOK, gin.H{
		"message":      "token refreshed",
		"access_token": accessTokenStr,
		"expires_in":   int(time.Until(accessExp).Seconds()),
	})
}
