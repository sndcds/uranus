package api

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/grains/grains_validation"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) ForgotPassword(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "forgot-password")
	ctx := gc.Request.Context()
	successMessage := "If an account exists, a reset link has been sent."

	var payload struct {
		Email   string `json:"email" binding:"required,email"`
		Referer string `json:"referer" binding:"required"`
	}

	if err := gc.ShouldBindJSON(&payload); err != nil {
		apiRequest.PayloadError()
		return
	}

	locale := app.NormalizeLocale(
		gc.DefaultQuery("lang", "en"),
	)

	query := fmt.Sprintf("SELECT uuid FROM %s.user WHERE email = $1", h.DbSchema)

	var userUuid string
	err := h.DbPool.QueryRow(ctx, query, payload.Email).Scan(&userUuid)
	if err != nil {
		apiRequest.SuccessNoData(http.StatusOK, successMessage)
		return
	}

	token, err := generateResetToken()
	if err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}

	//--------------------------------------------------------------------------
	// Store token in DB with expiry
	//--------------------------------------------------------------------------

	query = fmt.Sprintf(`
		INSERT INTO %s.password_reset (user_uuid, token, expires_at)
		VALUES ($1::uuid, $2, $3)`,
		h.DbSchema)

	expiryHours := 1
	_, err = h.DbPool.Exec(ctx, query, userUuid, token, time.Now().Add(time.Duration(expiryHours)*time.Hour))
	if err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}

	resetUrl := payload.Referer + "/app/reset-password?token=" + token

	//--------------------------------------------------------------------------
	// Email template
	//--------------------------------------------------------------------------

	layoutPath := fmt.Sprintf(
		"template/email/layout/%s.html",
		locale,
	)

	contentPath := fmt.Sprintf(
		"template/email/user-password-reset/%s.html",
		locale,
	)

	data := struct {
		Language    string
		ResetLink   string
		ExpiryHours int
	}{
		Language:    locale,
		ResetLink:   resetUrl,
		ExpiryHours: expiryHours,
	}

	subject, emailContent, err := app.RenderEmailTemplate(
		layoutPath,
		contentPath,
		data,
	)
	if err != nil {
		debugf("failed to render password reset email: %v", err)
		apiRequest.InternalServerError()
		return
	}

	go func() {
		if err := app.SendEmailWithTimeout(
			payload.Email,
			subject,
			emailContent,
			20*time.Second,
		); err != nil {
			debugf("password reset email failed: %v", err)
		}
	}()

	apiRequest.SuccessNoData(http.StatusOK, successMessage)
}

func (h *ApiHandler) ResetPassword(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "reset-password")
	ctx := gc.Request.Context()

	var req struct {
		Token       string `json:"token" binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}

	if err := gc.ShouldBindJSON(&req); err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}

	var userUuid string
	var expiresAt time.Time

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		query := fmt.Sprintf(`
			SELECT user_uuid, expires_at
			FROM %s.password_reset
			WHERE token = $1
			AND expires_at > NOW()`,
			h.DbSchema)

		err := tx.QueryRow(
			ctx,
			query,
			req.Token,
		).Scan(&userUuid, &expiresAt)

		if err != nil {
			return TxInternalError(err)
		}

		if err != nil {
			return TxInternalError(nil)
		}

		var userEmail string
		query = fmt.Sprintf(`SELECT email FROM %s.user WHERE uuid = $1::uuid`, h.DbSchema)
		err = tx.QueryRow(
			ctx, query,
			userUuid,
		).Scan(&userEmail)

		if err != nil {
			return TxInternalError(nil)
		}

		err = grains_validation.ValidatePassword(userEmail, req.NewPassword, 12)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusUnprocessableEntity,
				Err:  fmt.Errorf("(#1) password does not meet security requirements"),
			}
		}

		hashed, err := app.EncryptPassword(req.NewPassword)
		if err != nil {
			return TxInternalError(nil)
		}

		updateUserQuery := fmt.Sprintf(`UPDATE %s.user SET password_hash = $1 WHERE uuid = $2::uuid`, h.DbSchema)
		_, err = tx.Exec(ctx, updateUserQuery, hashed, userUuid)
		if err != nil {
			return TxInternalError(nil)
		}

		deleteQuery := fmt.Sprintf(`DELETE FROM %s.password_reset WHERE user_uuid = $1::uuid`, h.DbSchema)
		_, err = tx.Exec(ctx, deleteQuery, userUuid)
		if err != nil {
			return TxInternalError(nil)
		}

		return nil
	})

	if txErr != nil {
		debugf(txErr.Error())
		apiRequest.Error(txErr.Code, txErr.Error())
		return
	}

	apiRequest.SuccessNoData(http.StatusOK, "password reset successful.")
}

func generateResetToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
