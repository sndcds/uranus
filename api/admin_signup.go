package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) Signup(gc *gin.Context) {
	pool := h.DbPool

	langStr := gc.DefaultQuery("lang", "en")

	// Parse incoming JSON
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := gc.BindJSON(&req); err != nil || req.Email == "" || req.Password == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "email and password required"})
		return
	}

	// Validate
	if !app.IsValidEmail(req.Email) {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "invalid email"})
	}

	// Encrypt password
	passwordHash, err := app.EncryptPassword(req.Password)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encrypt password"})
		return
	}

	// Check if user already exists
	var exists bool
	checkQuery := fmt.Sprintf(
		"SELECT EXISTS(SELECT 1 FROM %s.user WHERE email_address = $1)",
		h.Config.DbSchema)
	err = pool.QueryRow(gc, checkQuery, req.Email).Scan(&exists)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	if exists {
		gc.JSON(http.StatusConflict, gin.H{"error": "user already exists"})
		return
	}

	// Insert new user
	var newUserId int
	insertQuery := fmt.Sprintf(`
		INSERT INTO %s.user (email_address, password_hash)
		VALUES ($1, $2)
		RETURNING id`,
		h.Config.DbSchema)

	err = pool.QueryRow(gc, insertQuery, req.Email, passwordHash).Scan(&newUserId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	// Generate token and send email to users
	signupExp := time.Now().Add(time.Duration(h.Config.AuthTokenExpirationTime) * time.Second)
	signupClaims := &app.Claims{
		UserId: newUserId,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(signupExp),
		},
	}
	signupToken := jwt.NewWithClaims(jwt.SigningMethodHS256, signupClaims)
	signupTokenString, err := signupToken.SignedString([]byte(h.Config.JwtSecret))
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	updateQuery := fmt.Sprintf(`UPDATE %s.user SET activate_token = $1 WHERE id = $2`, h.Config.DbSchema)

	_, err = pool.Exec(gc, updateQuery, signupTokenString, newUserId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	messageQuery := fmt.Sprintf(`SELECT template FROM %s.system_email_template WHERE context = 'activate-email' AND iso_639_1 = $1`, h.Config.DbSchema)
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

	linkUrl := gc.Request.Referer() + "app/activate/account?token=" + signupTokenString
	template = strings.Replace(template, "{{link}}", linkUrl, 1)

	go func() {
		sendEmailErr := sendEmail(req.Email, template)
		if sendEmailErr != nil {
			gc.JSON(http.StatusOK, gin.H{
				"message":    "Unable to send reset email.",
				"error_code": -1,
			})
		}
	}()

	// Respond success
	gc.JSON(http.StatusCreated, gin.H{
		"message": "user created successfully",
		"user_id": newUserId,
	})
}

func (h *ApiHandler) Activate(gc *gin.Context) {
	pool := h.DbPool

	var req struct {
		Token string `json:"token"`
	}
	if err := gc.BindJSON(&req); err != nil || req.Token == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "token required"})
		return
	}

	// Parse JWT token using the same signing method
	token, err := jwt.ParseWithClaims(req.Token, &app.Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(h.Config.JwtSecret), nil
	})
	if err != nil {
		gc.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
		return
	}

	// Extract claims
	claims, ok := token.Claims.(*app.Claims)
	if !ok || !token.Valid {
		gc.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	userId := claims.UserId

	// Query stored activation token
	var storedToken string
	query := fmt.Sprintf(`SELECT activate_token FROM %s.user WHERE id = $1`, h.Config.DbSchema)
	err = pool.QueryRow(gc, query, userId).Scan(&storedToken)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			gc.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		} else {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		}
		return
	}

	// Compare tokens
	if storedToken != req.Token {
		gc.JSON(http.StatusUnauthorized, gin.H{"error": "token mismatch"})
		return
	}

	// Activate account
	updateQuery := fmt.Sprintf(`UPDATE %s.user SET is_active = true, activate_token = NULL WHERE id = $1`, h.Config.DbSchema)
	if _, err := pool.Exec(gc, updateQuery, userId); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to activate user"})
		return
	}

	gc.JSON(http.StatusOK, gin.H{"message": "account successfully activated"})
}
