package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

func (h *ApiHandler) Login(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "login")

	var userCredentials model.UserCredentials

	// Parse credentials
	err := gc.ShouldBindJSON(&userCredentials)
	if err != nil {
		debugf("invalid login request: %v", err)
		apiRequest.Error(http.StatusBadRequest, "invalid request")
		return
	}

	if userCredentials.Email == "" || userCredentials.Password == "" {
		apiRequest.Error(http.StatusUnauthorized, "invalid email or password")
		return
	}

	email := strings.TrimSpace(userCredentials.Email)
	var user model.User
	query := fmt.Sprintf(
		`SELECT uuid, email, password_hash, first_name, last_name, display_name, locale, theme, is_active
		FROM %s.user WHERE email = $1`,
		h.DbSchema)
	err = h.DbPool.QueryRow(gc, query, email).Scan(
		&user.Uuid,
		&user.Email,
		&user.PasswordHash,
		&user.FirstName,
		&user.LastName,
		&user.DisplayName,
		&user.Locale,
		&user.Theme,
		&user.IsActive,
	)
	if err != nil {
		debugf(err.Error())
		apiRequest.Error(http.StatusUnauthorized, "invalid email or password")
		return
	}

	if !user.IsActive {
		apiRequest.Error(http.StatusUnauthorized, "invalid email or password")
		return
	}

	if user.PasswordHash == nil {
		apiRequest.Error(http.StatusUnauthorized, "invalid email or password")
		return
	}

	if app.ComparePasswords(*user.PasswordHash, userCredentials.Password) != nil {
		apiRequest.Error(http.StatusUnauthorized, "invalid email or password")
		return
	}

	now := time.Now()

	// Create access token
	accessExp := now.Add(time.Duration(h.Config.AuthTokenExpirationTime) * time.Second)
	accessClaims := &app.Claims{
		UserUuid:  user.Uuid,
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(accessExp),
		},
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenStr, err := accessToken.SignedString(app.UranusInstance.JwtKey)
	if err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}

	// Create refresh token
	refreshExp := now.Add(time.Duration(h.Config.RefreshTokenExpirationTime) * time.Second)
	refreshClaims := &app.Claims{
		UserUuid:  user.Uuid,
		TokenType: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(refreshExp),
		},
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenStr, err := refreshToken.SignedString(app.UranusInstance.JwtKey)
	if err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}

	apiRequest.Success(http.StatusOK, gin.H{
		"user_uuid":     user.Uuid,
		"display_name":  user.DisplayName,
		"first_name":    user.FirstName,
		"last_name":     user.LastName,
		"locale":        user.Locale,
		"theme":         user.Theme,
		"access_token":  accessTokenStr,
		"refresh_token": refreshTokenStr,
		"avatar_url":    app.GetAvatarURL(h.Config.BaseApiUrl, h.Config.ProfileImageDir, user.Uuid, 64),
	}, "login successful")
}

func (h *ApiHandler) Refresh(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "refresh access token")
	const refreshErrorMsg = "invalid refresh token"

	// Get token from Authorization header
	authHeader := gc.GetHeader("Authorization")
	parts := strings.Fields(authHeader)

	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		debugf("invalid refresh authorization header")
		apiRequest.Error(http.StatusUnauthorized, refreshErrorMsg)
		return
	}

	refreshToken := parts[1]

	// Parse token
	claims := &app.Claims{}
	tkn, err := jwt.ParseWithClaims(
		refreshToken,
		claims,
		func(token *jwt.Token) (any, error) {
			return app.UranusInstance.JwtKey, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil || !tkn.Valid {
		apiRequest.Error(http.StatusUnauthorized, refreshErrorMsg)
		return
	}

	if claims.TokenType != "refresh" {
		apiRequest.Error(http.StatusUnauthorized, refreshErrorMsg)
		return
	}

	// Query user and check if active

	var isActive bool

	query := fmt.Sprintf(
		`SELECT is_active FROM %s.user WHERE uuid = $1`,
		h.DbSchema,
	)

	err = h.DbPool.QueryRow(gc, query, claims.UserUuid).Scan(&isActive)
	if err != nil || !isActive {
		apiRequest.Error(http.StatusUnauthorized, refreshErrorMsg)
		return
	}

	now := time.Now()

	// Issue new access token
	accessExp := now.Add(time.Duration(h.Config.AuthTokenExpirationTime) * time.Second)
	newClaims := &app.Claims{
		UserUuid:  claims.UserUuid,
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(accessExp),
		},
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, newClaims)
	accessTokenStr, err := accessToken.SignedString(app.UranusInstance.JwtKey)
	if err != nil {
		debugf("failed to sign new access token for user_uuid=%s: %v", claims.UserUuid, err)
		apiRequest.InternalServerError()
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
