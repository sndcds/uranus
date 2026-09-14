package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

func (h *ApiHandler) Login(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "login")
	// Browsers must originate from the same allowlist as credentialed CORS.
	if (gc.GetHeader("Origin") != "" || gc.GetHeader("Referer") != "" || gc.GetHeader("Sec-Fetch-Site") != "") && !app.RequireAuthOrigin(gc) {
		return
	}

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
	err = h.DbPool.QueryRow(gc.Request.Context(), query, email).Scan(
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

	var pair *authTokenPair
	txErr := WithTransaction(gc.Request.Context(), h.DbPool, func(tx pgx.Tx) *ApiTxError {
		// Recheck under the same lock used for rotation and account changes.
		var active bool
		var currentHash string
		err := tx.QueryRow(gc.Request.Context(), fmt.Sprintf(
			`SELECT is_active, password_hash FROM %s."user" WHERE uuid = $1 FOR UPDATE`, h.DbSchema), user.Uuid).Scan(&active, &currentHash)
		if err != nil {
			return authDatabaseError(err)
		}
		if !active || currentHash != *user.PasswordHash {
			return invalidRefreshError()
		}
		pair, err = h.newAuthTokenPair(user.Uuid)
		if err != nil {
			return TxInternalError(err)
		}
		return h.insertRefreshToken(gc.Request.Context(), tx, pair, pair.refreshClaims.ID)
	})
	if txErr != nil {
		debugf("login transaction failed: %v", txErr)
		if txErr.Code == http.StatusUnauthorized {
			apiRequest.Error(http.StatusUnauthorized, "invalid email or password")
		} else {
			apiRequest.InternalServerError()
		}
		return
	}
	h.setAuthCookies(gc, pair)

	apiRequest.Success(http.StatusOK, gin.H{
		"user_uuid":     user.Uuid,
		"display_name":  user.DisplayName,
		"first_name":    user.FirstName,
		"last_name":     user.LastName,
		"locale":        user.Locale,
		"theme":         user.Theme,
		"access_token":  pair.accessToken,
		"refresh_token": pair.refreshToken,
		"avatar_url":    app.GetAvatarURL(h.Config.BaseApiUrl, h.Config.ProfileImageDir, user.Uuid, 64),
	}, "login successful")
}
