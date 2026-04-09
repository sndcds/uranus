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
	err := gc.BindJSON(&userCredentials)
	if err != nil {
		debugf(err.Error())
		apiRequest.Error(http.StatusUnauthorized, "invalid credentials")
		return
	}

	if userCredentials.Email == "" || userCredentials.Password == "" {
		debugf(err.Error())
		apiRequest.Error(http.StatusUnauthorized, "invalid email or password")
		return
	}

	var user model.User
	query := fmt.Sprintf(
		`SELECT uuid, email, password_hash, first_name, last_name, display_name, locale, theme, is_active
		FROM %s.user WHERE email = $1`,
		h.DbSchema)
	err = h.DbPool.QueryRow(gc, query, userCredentials.Email).Scan(
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
		apiRequest.Error(http.StatusUnauthorized, "login error")
		return
	}

	if !user.IsActive || app.ComparePasswords(*user.PasswordHash, userCredentials.Password) != nil {
		debugf(err.Error())
		apiRequest.Error(http.StatusUnauthorized, "login failed")
		return
	}

	// Create access token
	accessExp := time.Now().Add(time.Duration(h.Config.AuthTokenExpirationTime) * time.Second)
	accessClaims := &app.Claims{
		UserUuid: user.Uuid,
		RegisteredClaims: jwt.RegisteredClaims{
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
	refreshExp := time.Now().Add(7 * 24 * time.Hour)
	refreshClaims := &app.Claims{
		UserUuid: user.Uuid,
		RegisteredClaims: jwt.RegisteredClaims{
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
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		debugf("Invalid refresh header: %s", authHeader)
		apiRequest.Error(http.StatusUnauthorized, "failed")
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
		debugf("Invalid refresh token: %v", err)
		apiRequest.Error(http.StatusUnauthorized, "failed")
		return
	}

	// Issue new access token
	accessExp := time.Now().Add(time.Duration(h.Config.AuthTokenExpirationTime) * time.Second)
	newClaims := &app.Claims{
		UserUuid: claims.UserUuid,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExp),
		},
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, newClaims)
	accessTokenStr, err := accessToken.SignedString(app.UranusInstance.JwtKey)
	if err != nil {
		debugf("Failed to sign new access token for user_uuid=%s: %v", claims.UserUuid, err)
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
