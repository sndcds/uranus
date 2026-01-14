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
)

// TODO: Review code

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
	var userId int
	var emailAddress string
	var passwordHash string
	var displayName *string
	var firstName *string
	var lastName *string
	var locale *string
	var theme *string
	var isActive bool

	query := fmt.Sprintf(
		`SELECT id, email_address, password_hash, first_name, last_name, display_name, locale, theme, is_active
		FROM %s.user WHERE email_address = $1`,
		h.Config.DbSchema)
	err := h.DbPool.QueryRow(gc, query, credentials.Email).Scan(
		&userId,
		&emailAddress,
		&passwordHash,
		&firstName,
		&lastName,
		&displayName,
		&locale,
		&theme,
		&isActive,
	)
	if err != nil {
		gc.JSON(http.StatusUnauthorized, gin.H{"error": loginErrorMsg})
		return
	}

	if !isActive || app.ComparePasswords(passwordHash, credentials.Password) != nil {
		gc.JSON(http.StatusUnauthorized, gin.H{"error": loginErrorMsg})
		return
	}

	// Create access token
	accessExp := time.Now().Add(time.Duration(h.Config.AuthTokenExpirationTime) * time.Second)
	accessClaims := &app.Claims{
		UserId: userId,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExp),
		},
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenStr, err := accessToken.SignedString(app.UranusInstance.JwtKey)
	if err != nil {
		log.Printf("Failed to sign access token for user=%d: %v", userId, err)
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	// Create refresh token
	refreshExp := time.Now().Add(7 * 24 * time.Hour)
	refreshClaims := &app.Claims{
		UserId: userId,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshExp),
		},
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenStr, err := refreshToken.SignedString(app.UranusInstance.JwtKey)
	if err != nil {
		log.Printf("Failed to sign refresh token for user=%d: %v", userId, err)
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	// Return tokens
	gc.JSON(http.StatusOK, gin.H{
		"message":       "login successful",
		"user_id":       userId,
		"display_name":  displayName,
		"first_name":    firstName,
		"last_name":     lastName,
		"locale":        locale,
		"theme":         theme,
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
		return app.UranusInstance.JwtKey, nil
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
	accessTokenStr, err := accessToken.SignedString(app.UranusInstance.JwtKey)
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
