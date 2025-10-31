package api

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/smtp"
	"time"

	"github.com/gin-gonic/gin"
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

	db := h.DbPool
	ctx := gc.Request.Context()

	// Look up user
	query := fmt.Sprintf(
		"SELECT id FROM %s.user WHERE email_address = $1",
		h.Config.DbSchema,
	)

	var userID int
	err := db.QueryRow(ctx, query, req.EmailAddress).Scan(&userID)
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

	_, err = db.Exec(ctx, query, userID, token, time.Now().Add(1*time.Hour))
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save token"})
		return
	}

	// Build reset link
	resetURL := fmt.Sprintf(
		"%s?token=%s",
		h.Config.AuthResetPasswordUrl,
		token)
	go func() {
		sendResetEmail(req.EmailAddress, resetURL)
	}()

	gc.JSON(http.StatusOK, gin.H{"message": "If an account exists, a reset link has been sent."})
}

func (h *ApiHandler) ResetPassword(gc *gin.Context) {
	fmt.Println("Reset Password")
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
	fmt.Printf(query)
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
	fmt.Printf(updateUserQuery)

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

func sendResetEmail(to, link string) {
	from := app.Singleton.Config.AuthReplyEmailAddress
	password := app.Singleton.Config.AuthSmtpPassword
	smtpHost := app.Singleton.Config.AuthSmtpHost
	smtpPort := app.Singleton.Config.AuthSmtpPort

	// Message body (RFC 822 format)
	message := []byte("Subject: Hello from Go!\r\n" +
		"MIME-Version: 1.0\r\n" +
		"To: undisclosed-recipients:;\r\n" +
		"From: " + from + "\r\n" +
		"Content-Type: text/plain; charset=\"UTF-8\"\r\n" +
		"\r\n" +
		"Click the link below to reset your password (valid for 1 hour):\r\n" +
		link + "\r\n")

	// Authentication
	auth := smtp.PlainAuth("", from, password, smtpHost)
	addr := fmt.Sprintf("%s:%d", smtpHost, smtpPort)

	// Send the email
	err := smtp.SendMail(addr, auth, from, []string{to}, message)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
}
