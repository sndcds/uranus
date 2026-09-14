package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/grains/grains_uuid"
	"github.com/sndcds/uranus/app"
)

type authTokenPair struct {
	accessToken   string
	refreshToken  string
	accessClaims  *app.Claims
	refreshClaims *app.Claims
}

func signAuthToken(userUUID, tokenType, id string, now time.Time, lifetime int) (string, *app.Claims, error) {
	if app.UranusInstance == nil || len(app.UranusInstance.JwtKey) == 0 {
		return "", nil, errors.New("JWT signing key is not configured")
	}
	claims := &app.Claims{UserUuid: userUUID, TokenType: tokenType, RegisteredClaims: jwt.RegisteredClaims{
		ID: id, IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(lifetime) * time.Second)),
	}}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(app.UranusInstance.JwtKey)
	return token, claims, err
}

func (h *ApiHandler) newAuthTokenPair(userUUID string) (*authTokenPair, error) {
	// Bound old configurations as well as the default to a short access lifetime.
	accessLifetime := min(h.Config.AuthTokenExpirationTime, 900)
	refreshLifetime := h.Config.RefreshTokenExpirationTime
	if accessLifetime <= 0 || refreshLifetime <= accessLifetime {
		return nil, errors.New("invalid authentication token lifetimes")
	}
	id, err := grains_uuid.Uuidv7String()
	if err != nil {
		return nil, err
	}
	now := time.Now()
	pair := &authTokenPair{}
	pair.accessToken, pair.accessClaims, err = signAuthToken(userUUID, app.AccessTokenType, "", now, accessLifetime)
	if err != nil {
		return nil, err
	}
	pair.refreshToken, pair.refreshClaims, err = signAuthToken(userUUID, app.RefreshTokenType, id, now, refreshLifetime)
	if err != nil {
		return nil, err
	}
	return pair, nil
}

func invalidRefreshError() *ApiTxError {
	return &ApiTxError{Code: http.StatusUnauthorized, Message: "invalid refresh token"}
}

func authDatabaseError(err error) *ApiTxError {
	if errors.Is(err, pgx.ErrNoRows) {
		return invalidRefreshError()
	}
	return TxInternalError(err)
}

func (h *ApiHandler) insertRefreshToken(ctx context.Context, tx pgx.Tx, pair *authTokenPair, familyUUID string) *ApiTxError {
	c := pair.refreshClaims
	_, err := tx.Exec(ctx, fmt.Sprintf(`INSERT INTO %s.refresh_token
  (jti, user_uuid, family_uuid, created_at, expires_at) VALUES ($1, $2, $3, $4, $5)`, h.DbSchema),
		c.ID, c.UserUuid, familyUUID, c.IssuedAt.Time, c.ExpiresAt.Time)
	if err != nil {
		return TxInternalError(err)
	}
	return nil
}

func (h *ApiHandler) revokeRefreshFamily(ctx context.Context, tx pgx.Tx, userUUID, familyUUID string) *ApiTxError {
	_, err := tx.Exec(ctx, fmt.Sprintf(`UPDATE %s.refresh_token SET revoked_at = CURRENT_TIMESTAMP
  WHERE user_uuid = $1 AND family_uuid = $2 AND revoked_at IS NULL`, h.DbSchema), userUUID, familyUUID)
	if err != nil {
		return TxInternalError(err)
	}
	return nil
}

func (h *ApiHandler) authCookie(gc *gin.Context, name, value, path string, expires time.Time, maxAge int) {
	http.SetCookie(gc.Writer, &http.Cookie{
		Name: name, Value: value, Path: path, Expires: expires, MaxAge: maxAge,
		HttpOnly: true, Secure: !h.Config.DevMode, SameSite: http.SameSiteLaxMode,
	})
}

func (h *ApiHandler) setAuthCookies(gc *gin.Context, pair *authTokenPair) {
	gc.Header("Cache-Control", "no-store")
	h.authCookie(gc, "access_token", pair.accessToken, "/api", pair.accessClaims.ExpiresAt.Time,
		max(1, int(time.Until(pair.accessClaims.ExpiresAt.Time).Seconds())))
	h.authCookie(gc, "refresh_token", pair.refreshToken, "/api/admin", pair.refreshClaims.ExpiresAt.Time,
		max(1, int(time.Until(pair.refreshClaims.ExpiresAt.Time).Seconds())))
}

func (h *ApiHandler) clearAuthCookies(gc *gin.Context) {
	gc.Header("Cache-Control", "no-store")
	h.authCookie(gc, "access_token", "", "/api", time.Unix(1, 0), -1)
	h.authCookie(gc, "refresh_token", "", "/api/admin", time.Unix(1, 0), -1)
}

// Prefer the refresh cookie. The existing dashboard's Bearer transport remains
// supported when no cookie is present; both use the same validation and DB state.
func refreshClaims(gc *gin.Context) (*app.Claims, bool) {
	token, err := gc.Cookie("refresh_token")
	if err == nil && token != "" {
		if !app.RequireAuthOrigin(gc) {
			return nil, false
		}
	} else {
		parts := strings.Fields(gc.GetHeader("Authorization"))
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			token = parts[1]
		}
	}
	claims, err := app.ParseJWT(token)
	if err != nil || claims.TokenType != app.RefreshTokenType || !app.ValidUUID(claims.UserUuid) || !app.ValidUUID(claims.ID) {
		return nil, true
	}
	return claims, true
}

func (h *ApiHandler) RegisterSessionRoutes(router *gin.Engine) {
	router.POST("/api/admin/refresh", h.Refresh)
	router.POST("/api/admin/logout", h.Logout)
}

func (h *ApiHandler) Refresh(gc *gin.Context) {
	request := grains_api.NewRequest(gc, "refresh access token")
	gc.Header("Cache-Control", "no-store")
	claims, proceed := refreshClaims(gc)
	if !proceed {
		return
	}
	if claims == nil {
		request.Error(http.StatusUnauthorized, "invalid refresh token")
		return
	}
	ctx := gc.Request.Context()
	var pair *authTokenPair
	reused := false
	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		// Serialize rotation, reuse revocation and logout for this user. The stable
		// user row prevents races between different generations of the same family.
		var active bool
		err := tx.QueryRow(ctx, fmt.Sprintf(`SELECT is_active FROM %s."user" WHERE uuid = $1 FOR UPDATE`, h.DbSchema), claims.UserUuid).Scan(&active)
		if err != nil {
			return authDatabaseError(err)
		}
		if !active {
			return invalidRefreshError()
		}
		var familyUUID string
		var revoked, unexpired bool
		err = tx.QueryRow(ctx, fmt.Sprintf(`SELECT family_uuid, revoked_at IS NOT NULL, expires_at > clock_timestamp()
   FROM %s.refresh_token WHERE jti = $1 AND user_uuid = $2`, h.DbSchema), claims.ID, claims.UserUuid).Scan(&familyUUID, &revoked, &unexpired)
		if err != nil {
			return authDatabaseError(err)
		}
		if revoked {
			reused = true
			// Commit revocation before returning HTTP 401; an error here would roll it back.
			return h.revokeRefreshFamily(ctx, tx, claims.UserUuid, familyUUID)
		}
		if !unexpired || !claims.ExpiresAt.Time.After(time.Now()) {
			return invalidRefreshError()
		}
		pair, err = h.newAuthTokenPair(claims.UserUuid)
		if err != nil {
			return TxInternalError(err)
		}
		_, err = tx.Exec(ctx, fmt.Sprintf(`UPDATE %s.refresh_token SET revoked_at = CURRENT_TIMESTAMP WHERE jti = $1`, h.DbSchema), claims.ID)
		if err != nil {
			return TxInternalError(err)
		}
		return h.insertRefreshToken(ctx, tx, pair, familyUUID)
	})
	if txErr != nil {
		debugf("refresh transaction failed: %v", txErr)
		if txErr.Code == http.StatusUnauthorized {
			request.Error(http.StatusUnauthorized, "invalid refresh token")
		} else {
			request.InternalServerError()
		}
		return
	}
	if reused {
		request.Error(http.StatusUnauthorized, "invalid refresh token")
		return
	}
	h.setAuthCookies(gc, pair)
	gc.Header("Authorization", "Bearer "+pair.accessToken)
	gc.JSON(http.StatusOK, gin.H{
		"message": "token refreshed", "access_token": pair.accessToken, "refresh_token": pair.refreshToken,
		"expires_in": int(time.Until(pair.accessClaims.ExpiresAt.Time).Seconds()),
	})
}

func (h *ApiHandler) Logout(gc *gin.Context) {
	request := grains_api.NewRequest(gc, "logout")
	gc.Header("Cache-Control", "no-store")
	claims, proceed := refreshClaims(gc)
	if !proceed {
		return
	}
	if claims == nil {
		h.clearAuthCookies(gc)
		request.Error(http.StatusUnauthorized, "invalid refresh token")
		return
	}
	ctx := gc.Request.Context()
	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		var userUUID string
		err := tx.QueryRow(ctx, fmt.Sprintf(`SELECT uuid FROM %s."user" WHERE uuid = $1 FOR UPDATE`, h.DbSchema), claims.UserUuid).Scan(&userUUID)
		if err != nil {
			return authDatabaseError(err)
		}
		var familyUUID string
		err = tx.QueryRow(ctx, fmt.Sprintf(`SELECT family_uuid FROM %s.refresh_token WHERE jti = $1 AND user_uuid = $2`, h.DbSchema), claims.ID, claims.UserUuid).Scan(&familyUUID)
		if err != nil {
			return authDatabaseError(err)
		}
		return h.revokeRefreshFamily(ctx, tx, userUUID, familyUUID)
	})
	if txErr != nil {
		debugf("logout transaction failed: %v", txErr)
		if txErr.Code == http.StatusUnauthorized {
			h.clearAuthCookies(gc)
			request.Error(http.StatusUnauthorized, "invalid refresh token")
		} else {
			request.InternalServerError()
		}
		return
	}
	h.clearAuthCookies(gc)
	request.SuccessNoData(http.StatusOK, "logout successful")
}
