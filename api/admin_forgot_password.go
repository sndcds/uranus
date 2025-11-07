package api

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/smtp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) ForgotPassword(gc *gin.Context) {
	var req struct {
		EmailAddress string `json:"email"`
	}
	if err := gc.ShouldBindJSON(&req); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	pool := h.DbPool
	ctx := gc.Request.Context()

	// Look up user
	query := fmt.Sprintf(
		"SELECT id FROM %s.user WHERE email_address = $1",
		h.Config.DbSchema,
	)

	var userID int
	err := pool.QueryRow(ctx, query, req.EmailAddress).Scan(&userID)
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

	_, err = pool.Exec(ctx, query, userID, token, time.Now().Add(1*time.Hour))
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save token"})
		return
	}

	resetUrl := gc.Request.Referer() + "app/reset-password?token=" + token

	langStr := "en" // TODO: Which language?
	messageQuery := fmt.Sprintf(`SELECT template FROM %s.system_email_template WHERE context = 'reset-email' AND iso_639_1 = $1`, h.Config.DbSchema)
	_, err = pool.Exec(gc, messageQuery, langStr)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}
	var template string
	err = pool.QueryRow(gc, messageQuery, langStr).Scan(&template)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			gc.JSON(http.StatusNotFound, gin.H{"error": "email template not found"})
		} else {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get email template"})
		}
		return
	}
	emailContent := strings.Replace(template, "{{link}}", resetUrl, 1)
	go func() {
		sendEmailErr := sendEmail(req.EmailAddress, emailContent)
		if sendEmailErr != nil {
			gc.JSON(http.StatusOK, gin.H{
				"message":    "Unable to send reset email.",
				"error_code": -1,
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

	db := h.DbPool
	ctx := gc.Request.Context()

	var userId int
	var expiresAt time.Time
	var used bool

	query := fmt.Sprintf(`SELECT user_id, expires_at, used FROM %s.password_reset WHERE token = $1`,
		h.Config.DbSchema)
	err := db.QueryRow(
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
	tx, _ := db.Begin(ctx)
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

func sendEmail(to, htmlContent string) error {
	from := app.Singleton.Config.AuthReplyEmailAddress
	userName := app.Singleton.Config.AuthSmtpLogin
	password := app.Singleton.Config.AuthSmtpPassword
	smtpHost := app.Singleton.Config.AuthSmtpHost
	smtpPort := app.Singleton.Config.AuthSmtpPort // int

	// Build MIME email message with HTML content
	message := []byte("Subject: Reset your password\r\n" +
		"MIME-Version: 1.0\r\n" +
		"To: " + to + "\r\n" +
		"From: " + from + "\r\n" +
		"Content-Type: text/html; charset=\"UTF-8\"\r\n" +
		"\r\n" +
		htmlContent + "\r\n")

	auth := smtp.PlainAuth("", userName, password, smtpHost)
	addr := fmt.Sprintf("%s:%d", smtpHost, smtpPort)

	fmt.Println("auth:", auth)
	fmt.Println("addr:", addr)

	err := smtp.SendMail(addr, auth, userName, []string{to}, message)
	if err != nil {
		fmt.Println("err:", err.Error())
		return fmt.Errorf("unable to send email: %s", err.Error())
	}

	return nil
}
