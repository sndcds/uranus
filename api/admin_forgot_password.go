package api

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/uranus/app"
	"golang.org/x/net/idna"
)

// TODO: Review code

func (h *ApiHandler) ForgotPassword(gc *gin.Context) {
	var req struct {
		EmailAddress string `json:"email"`
	}
	if err := gc.ShouldBindJSON(&req); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	ctx := gc.Request.Context()
	lang := gc.DefaultQuery("lang", "en")

	// Look up user
	query := fmt.Sprintf(
		"SELECT id FROM %s.user WHERE email_address = $1",
		h.Config.DbSchema,
	)

	var userID int
	err := h.DbPool.QueryRow(ctx, query, req.EmailAddress).Scan(&userID)
	if err != nil {
		// Always respond the same way to avoid leaking info
		gc.JSON(http.StatusOK, gin.H{"message": "If an account exists, a reset link has been sent."})
		return
	}

	// Generate a token
	token, err := generateResetToken()
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Store token in DB with expiry
	query = fmt.Sprintf(`
		INSERT INTO %s.password_reset (user_id, token, expires_at)
		VALUES ($1, $2, $3)`,
		h.Config.DbSchema)

	expiryHour := 1
	_, err = h.DbPool.Exec(ctx, query, userID, token, time.Now().Add(time.Duration(expiryHour)*time.Hour))
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save token"})
		return
	}

	resetUrl := gc.Request.Referer() + "app/reset-password?token=" + token

	messageQuery := fmt.Sprintf(`SELECT subject, template FROM %s.system_email_template WHERE context = 'reset-email' AND iso_639_1 = $1`, h.Config.DbSchema)
	_, err = h.DbPool.Exec(gc, messageQuery, lang)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate email"})
		return
	}
	var subject string
	var template string
	err = h.DbPool.QueryRow(gc, messageQuery, lang).Scan(&subject, &template)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			gc.JSON(http.StatusNotFound, gin.H{"error": "email template not found"})
		} else {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get email template"})
		}
		return
	}
	emailContent := strings.Replace(template, "{{link}}", resetUrl, -1)
	emailContent = strings.Replace(emailContent, "{{expiry_hours}}", strconv.Itoa(expiryHour), -1)
	go func() {
		sendEmailErr := sendEmail(req.EmailAddress, subject, emailContent)
		if sendEmailErr != nil {
			gc.JSON(http.StatusBadRequest, gin.H{
				"message": "Unable to send reset email.",
			})
		}
	}()

	gc.JSON(http.StatusOK, gin.H{
		"message":    "If an account exists, a reset link has been sent.",
		"error_code": 0,
	})
}

func (h *ApiHandler) ResetPassword(gc *gin.Context) {
	var req struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}
	if err := gc.ShouldBindJSON(&req); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	ctx := gc.Request.Context()

	var userId int
	var expiresAt time.Time
	var used bool

	query := fmt.Sprintf(`SELECT user_id, expires_at, used FROM %s.password_reset WHERE token = $1`,
		h.Config.DbSchema)
	err := h.DbPool.QueryRow(
		ctx,
		query,
		req.Token).Scan(&userId, &expiresAt, &used)
	//	if err != nil || used || time.Now().UTC().After(expiresAt) {
	if err != nil || used {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired token"})
		return
	}

	// Hash the password
	hashed, err := app.EncryptPassword(req.NewPassword)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Update user password and mark token used
	tx, _ := h.DbPool.Begin(ctx)
	defer func() { _ = tx.Rollback(ctx) }()

	// Update user's password
	updateUserQuery := fmt.Sprintf(
		`UPDATE %s.user SET password_hash = $1 WHERE id = $2`,
		h.Config.DbSchema,
	)

	_, err = tx.Exec(ctx, updateUserQuery, hashed, userId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	// Mark the reset token as used
	updateTokenQuery := fmt.Sprintf(
		`UPDATE %s.password_reset SET used = TRUE WHERE token = $1`,
		h.Config.DbSchema,
	)
	_, err = tx.Exec(ctx, updateTokenQuery, req.Token)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark token used"})
		return
	}

	tx.Commit(ctx)
	gc.JSON(http.StatusOK, gin.H{"message": "Password reset successful"})
}

func generateResetToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func sendEmail(to, subject string, htmlContent string) error {
	from := app.UranusInstance.Config.AuthReplyEmailAddress
	userName := app.UranusInstance.Config.AuthSmtpLogin
	password := app.UranusInstance.Config.AuthSmtpPassword
	smtpHost := app.UranusInstance.Config.AuthSmtpHost
	smtpPort := app.UranusInstance.Config.AuthSmtpPort // int

	asciiFrom, err := encodeEmailAddress(from)
	if err != nil {
		return fmt.Errorf("unable to send email: %s", err.Error())
	}

	asciiTo, err := encodeEmailAddress(to)
	if err != nil {
		return fmt.Errorf("unable to send email: %s", err.Error())
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
		return fmt.Errorf("unable to send email: %s", err.Error())
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
