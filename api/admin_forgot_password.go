package api

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
	"golang.org/x/net/idna"
)

func (h *ApiHandler) ForgotPassword(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "forgot-password")

	var req struct {
		EmailAddress string `json:"email" binding:"required"`
	}

	if err := gc.ShouldBindJSON(&req); err != nil {
		apiRequest.Error(http.StatusBadRequest, "invalid request")
		return
	}

	ctx := gc.Request.Context()
	lang := gc.DefaultQuery("lang", "en")

	query := fmt.Sprintf("SELECT uuid FROM %s.user WHERE email = $1", h.DbSchema)

	var userUuid string
	err := h.DbPool.QueryRow(ctx, query, req.EmailAddress).Scan(&userUuid)
	if err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}

	token, err := generateResetToken()
	if err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}

	// Store token in DB with expiry
	query = fmt.Sprintf(`
		INSERT INTO %s.password_reset (user_uuid, token, expires_at)
		VALUES ($1::uuid, $2, $3)`,
		h.DbSchema)

	expiryHour := 1
	_, err = h.DbPool.Exec(ctx, query, userUuid, token, time.Now().Add(time.Duration(expiryHour)*time.Hour))
	if err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}

	resetUrl := gc.Request.Referer() + "app/reset-password?token=" + token

	messageQuery := fmt.Sprintf(`SELECT subject, template FROM %s.system_email_template WHERE context = 'reset-email' AND iso_639_1 = $1`, h.DbSchema)
	_, err = h.DbPool.Exec(gc, messageQuery, lang)
	if err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}
	var subject string
	var template string
	err = h.DbPool.QueryRow(gc, messageQuery, lang).Scan(&subject, &template)
	if err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}
	emailContent := strings.Replace(template, "{{link}}", resetUrl, -1)
	emailContent = strings.Replace(emailContent, "{{expiry_hours}}", strconv.Itoa(expiryHour), -1)

	// Create a context with timeout for sending email
	emailCtx, cancel := context.WithTimeout(ctx, 10*time.Second) // 10s timeout
	defer cancel()

	err = sendEmailWithContext(emailCtx, req.EmailAddress, subject, emailContent)
	if err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}

	apiRequest.SuccessNoData(http.StatusOK, "If an account exists, a reset link has been sent.")
}

func (h *ApiHandler) ResetPassword(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "reset-password")

	debugf("ResetPassword")

	var req struct {
		Token       string `json:"token" binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}
	if err := gc.ShouldBindJSON(&req); err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}

	ctx := gc.Request.Context()

	var userUuid string
	var expiresAt time.Time
	var used bool

	query := fmt.Sprintf(`SELECT user_uuid, expires_at, used FROM %s.password_reset WHERE token = $1`, h.DbSchema)
	debugf(query)
	err := h.DbPool.QueryRow(
		ctx,
		query,
		req.Token).Scan(&userUuid, &expiresAt, &used)
	//	if err != nil || used || time.Now().UTC().After(expiresAt) {
	if err != nil {
		debugf(err.Error())
		apiRequest.Error(http.StatusBadRequest, "invalid or expired token")
		return
	}
	if used {
		debugf("used!")
		apiRequest.Error(http.StatusBadRequest, "invalid or expired token")
		return
	}

	// Hash the password
	hashed, err := app.EncryptPassword(req.NewPassword)
	if err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {

		// Update user's password
		updateUserQuery := fmt.Sprintf(`UPDATE %s.user SET password_hash = $1 WHERE uuid = $2::uuid`, h.DbSchema)
		_, err = tx.Exec(ctx, updateUserQuery, hashed, userUuid)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  err,
			}
		}

		// Mark the reset token as used
		updateTokenQuery := fmt.Sprintf(`UPDATE %s.password_reset SET used = TRUE WHERE token = $1`, h.DbSchema)
		_, err = tx.Exec(ctx, updateTokenQuery, req.Token)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  err,
			}
		}

		return nil
	})
	if txErr != nil {
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

func sendEmailWithTimeout(to, subject, htmlContent string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	errCh := make(chan error, 1)

	go func() {
		errCh <- sendEmail(to, subject, htmlContent)
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("send email timeout: %w", ctx.Err())

	case err := <-errCh:
		return err
	}
}

func sendEmail(to, subject string, htmlContent string) error {
	from := app.UranusInstance.Config.AuthReplyEmail
	userName := app.UranusInstance.Config.AuthSmtpLogin
	password := app.UranusInstance.Config.AuthSmtpPassword
	smtpHost := app.UranusInstance.Config.AuthSmtpHost
	smtpPort := app.UranusInstance.Config.AuthSmtpPort // int

	debugf("sendEmail from: %s", from)
	asciiFrom, err := encodeEmailAddress(from)
	if err != nil {
		return fmt.Errorf("unable to send email 1: %s", err.Error())
	}

	debugf("sendEmail to: %s", to)
	asciiTo, err := encodeEmailAddress(to)
	if err != nil {
		return fmt.Errorf("unable to send email 2: %s", err.Error())
	}

	// Encode subject in Base64 for UTF-8
	encodedSubject := fmt.Sprintf("=?UTF-8?B?%s?=", base64.StdEncoding.EncodeToString([]byte(subject)))

	message := []byte(
		"Subject: " + encodedSubject + "\r\n" +
			"MIME-Version: 1.0\r\n" +
			"To: " + asciiTo + "\r\n" +
			"From: " + asciiFrom + "\r\n" +
			"Content-Type: text/html; charset=\"UTF-8\"\r\n" +
			"\r\n" +
			htmlContent + "\r\n")

	auth := smtp.PlainAuth("", userName, password, smtpHost)
	addr := fmt.Sprintf("%s:%d", smtpHost, smtpPort)

	err = smtp.SendMail(addr, auth, userName, []string{to}, message)
	if err != nil {
		return fmt.Errorf("unable to send email 3: %s", err.Error())
	}

	return nil
}

// Encode an email address for SMTP
func encodeEmailAddress(email string) (string, error) {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid email: %s", email)
	}

	local := parts[0]  // user
	domain := parts[1] // domain

	asciiDomain, err := idna.ToASCII(domain)
	if err != nil {
		return "", err
	}

	return local + "@" + asciiDomain, nil
}
