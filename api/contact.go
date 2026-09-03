package api

import (
	"fmt"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
)

const (
	contactRateLimit  = 5
	contactRatePeriod = time.Hour
)

func (h *ApiHandler) Contact(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "contact")
	ctx := gc.Request.Context()

	clientIP := gc.ClientIP()
	ipHash := HashIPAddress(
		clientIP,
		app.UranusInstance.Config.ContactSecret,
	)

	email := strings.TrimSpace(gc.PostForm("email"))
	message := strings.TrimSpace(gc.PostForm("message"))
	website := strings.TrimSpace(gc.PostForm("website"))

	if email == "" {
		apiRequest.Required("email is required")
		return
	}

	if _, err := mail.ParseAddress(email); err != nil {
		apiRequest.Error(http.StatusBadRequest, "email is invalid")
		return
	}

	if message == "" {
		apiRequest.Required("message is required")
		return
	}

	if len(message) < app.UranusInstance.Config.ContactMassageMinLength {
		apiRequest.Error(
			http.StatusBadRequest,
			"message needs to be at least 10 characters",
		)
		return
	}

	if len(message) > app.UranusInstance.Config.ContactMassageMaxLength {
		apiRequest.Error(
			http.StatusBadRequest,
			"message is too long",
		)
		return
	}

	// Honeypot.
	if website != "" {
		// Do not reveal that the honeypot was triggered.
		apiRequest.SuccessNoData(
			http.StatusCreated,
			"message received successfully",
		)
		return
	}

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {

		// Check rate limit for this IP
		ipRateLimitQuery := fmt.Sprintf(`
			SELECT COUNT(*)
			FROM %s.contact_message
			WHERE ip_hash = $1
			AND created_at >= NOW() - ($2 * INTERVAL '1 minute')
		`, h.DbSchema)

		var ipCount int64

		err := tx.QueryRow(
			ctx,
			ipRateLimitQuery,
			ipHash,
			app.UranusInstance.Config.ContactRatePeriodMinutes,
		).Scan(&ipCount)

		if err != nil {
			return TxInternalError(err)
		}

		if ipCount >= app.UranusInstance.Config.ContactRateLimit {
			return &ApiTxError{
				Code:    http.StatusTooManyRequests,
				Message: "too many contact messages",
			}
		}

		// Check rate limit for this email address
		emailRateLimitQuery := fmt.Sprintf(`
			SELECT COUNT(*)
			FROM %s.contact_message
			WHERE LOWER(email) = LOWER($1)
			AND created_at >= NOW() - ($2 * INTERVAL '1 minute')
		`, h.DbSchema)

		var emailCount int64

		err = tx.QueryRow(
			ctx,
			emailRateLimitQuery,
			email,
			app.UranusInstance.Config.ContactRatePeriodMinutes,
		).Scan(&emailCount)

		if err != nil {
			return TxInternalError(err)
		}

		if emailCount >= app.UranusInstance.Config.ContactEmailRateLimit {
			return &ApiTxError{
				Code:    http.StatusTooManyRequests,
				Message: "too many contact messages",
			}
		}

		// Store contact message
		insertQuery := fmt.Sprintf(`
		INSERT INTO %s.contact_message (
			ip_hash,
			email,
			message
		)
		VALUES (
			$1,
			$2,
			$3
		)
	`, h.DbSchema)

		_, err = tx.Exec(
			ctx,
			insertQuery,
			ipHash,
			email,
			message,
		)

		if err != nil {
			return TxInternalError(err)
		}

		return nil
	})

	if txErr != nil {
		debugf(txErr.Error())
		apiRequest.Error(txErr.Code, txErr.Error())
		return
	}

	apiRequest.SuccessNoData(
		http.StatusCreated,
		"message received successfully",
	)
}
