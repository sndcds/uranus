package api_admin

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

func ForgotPasswordHandler(gc *gin.Context) {
	var req struct {
		EmailAddress string `json:"email"`
	}
	if err := gc.ShouldBindJSON(&req); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	db := app.Singleton.MainDbPool
	ctx := gc.Request.Context()

	// Look up user
	query := fmt.Sprintf(
		"SELECT id FROM %s.user WHERE email_address = $1",
		app.Singleton.Config.DbSchema,
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
		app.Singleton.Config.DbSchema)

	_, err = db.Exec(ctx, query, userID, token, time.Now().Add(1*time.Hour))
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save token"})
		return
	}

	// Build reset link
	resetURL := fmt.Sprintf("https://uranus.oklabflensburg.de/api/admin/reset-password?token=%s", token)

	// Send the email
	go func() {
		sendTestMail(req.EmailAddress, resetURL)
	}()

	gc.JSON(http.StatusOK, gin.H{"message": "If an account exists, a reset link has been sent."})
}

func ResetPasswordHandler(gc *gin.Context) {
	fmt.Println("Reset Password")
	var req struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}
	if err := gc.ShouldBindJSON(&req); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	db := app.Singleton.MainDbPool
	ctx := gc.Request.Context()

	var userId int
	var expiresAt time.Time
	var used bool

	query := fmt.Sprintf(`SELECT user_id, expires_at, used FROM uranus.password_reset WHERE token = $1`,
		app.Singleton.Config.DbSchema)
	err := db.QueryRow(
		ctx,
		query,
		req.Token).Scan(&userId, &expiresAt, &used)

	if err != nil || used || time.Now().After(expiresAt) {
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
	defer tx.Rollback(ctx)

	// Update user's password
	updateUserQuery := fmt.Sprintf(
		`UPDATE %s.user SET password_hash = $1 WHERE id = $2`,
		app.Singleton.Config.DbSchema,
	)
	_, err = tx.Exec(ctx, updateUserQuery, hashed, userId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	// Mark the reset token as used
	updateTokenQuery := fmt.Sprintf(
		`UPDATE %s.password_reset SET used = TRUE WHERE token = $1`,
		app.Singleton.Config.DbSchema,
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

func sendTestMail(to, link string) {
	from := "oklab_noreply@grain.one"
	password := app.Singleton.Config.SmtpPassword

	// Your SMTP server configuration
	smtpHost := app.Singleton.Config.SmtpHost
	smtpPort := app.Singleton.Config.SmtpPort

	// Message body (RFC 822 format)
	message := []byte("Subject: Hello from Go!\r\n" +
		"MIME-Version: 1.0\r\n" +
		"To: undisclosed-recipients:;\r\n" +
		"From: roald@grain.one\r\n" +
		"\r\n" +
		"Content-Type: text/plain; charset=\"UTF-8\"\r\n\r\n" +
		"Click the link below to reset your password (valid for 1 hour):\r\n" +
		link + "\r\n")

	fmt.Println("message: ", string(message))
	fmt.Println("to: ", to)

	// Authentication
	auth := smtp.PlainAuth("", from, password, smtpHost)
	addr := fmt.Sprintf("%s:%d", smtpHost, smtpPort)

	// Send the email
	err := smtp.SendMail(addr, auth, from, []string{to}, message)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Email sent successfully!")
}

func sendResetEmail(to, link string) error {
	from := app.Singleton.Config.AuthReplyEmailAddress
	password := app.Singleton.Config.SmtpPassword
	smtpHost := app.Singleton.Config.SmtpHost
	smtpPort := app.Singleton.Config.SmtpPort

	message := []byte("Subject: Reset your password\r\n" +
		"From: " + from + "\r\n" +
		"To: " + to + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=\"UTF-8\"\r\n\r\n" +
		"Click the link below to reset your password (valid for 1 hour):\r\n" +
		link + "\r\n")

	auth := smtp.PlainAuth("", from, password, smtpHost)
	addr := fmt.Sprintf("%s:%d", smtpHost, smtpPort)

	err := smtp.SendMail(addr, auth, from, []string{to}, message)
	if err != nil {
		fmt.Println("Error sending email:", err)
		return err
	}

	fmt.Println("Email sent successfully!")
	return nil
}
