package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/grains/grains_uuid"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

// Permission to use endpoint checked, 2026-01-11, Roald

func (h *ApiHandler) Signup(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "signup")
	lang := gc.DefaultQuery("lang", "en")

	// Incoming JSON
	var userCredentials model.UserCredentials

	err := gc.BindJSON(&userCredentials)
	if err != nil || userCredentials.Email == "" || userCredentials.Password == "" {
		apiRequest.Error(http.StatusBadRequest, "email and password required")
		return
	}

	// Validate
	if !app.IsValidEmail(userCredentials.Email) {
		apiRequest.Error(http.StatusBadRequest, "invalid email")
		return
	}

	// Encrypt password
	passwordHash, err := app.EncryptPassword(userCredentials.Password)
	if err != nil {
		apiRequest.Error(http.StatusInternalServerError, "failed to encrypt password")
		return
	}

	// Check if user already exists
	var exists bool
	checkQuery := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s.user WHERE email = $1)", h.DbSchema)
	err = h.DbPool.QueryRow(gc, checkQuery, userCredentials.Email).Scan(&exists)
	if err != nil {
		apiRequest.Error(http.StatusInternalServerError, "database error")
		return
	}
	if exists {
		apiRequest.Error(http.StatusConflict, "user already exists")
		return
	}

	// Insert new user
	userUuid, err := grains_uuid.Uuidv7String()
	if err != nil {
		apiRequest.Error(http.StatusInternalServerError, "uuid creation error")
		return
	}

	insertQuery := fmt.Sprintf(`INSERT INTO %s.user (uuid, email, password_hash) VALUES ($1, $2, $3)`, h.DbSchema)
	_, err = h.DbPool.Exec(gc, insertQuery, userUuid, userCredentials.Email, passwordHash)
	if err != nil {
		apiRequest.Error(http.StatusInternalServerError, "failed to create user")
		return
	}

	// Generate token and send email to users
	expiryHour := 1
	signupExp := time.Now().Add(time.Duration(expiryHour) * time.Hour)
	signupClaims := &app.Claims{
		UserUuid: userUuid,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(signupExp),
		},
	}
	signupToken := jwt.NewWithClaims(jwt.SigningMethodHS256, signupClaims)
	signupTokenString, err := signupToken.SignedString([]byte(h.Config.JwtSecret))
	if err != nil {
		apiRequest.Error(http.StatusInternalServerError, "failed to generate token")
		return
	}

	updateQuery := fmt.Sprintf(`UPDATE %s.user SET activate_token = $1 WHERE uuid = $2`, h.DbSchema)
	_, err = h.DbPool.Exec(gc, updateQuery, signupTokenString, userUuid)
	if err != nil {
		apiRequest.Error(http.StatusInternalServerError, "failed to create user")
		return
	}

	messageQuery := fmt.Sprintf(`SELECT subject, template FROM %s.system_email_template WHERE context = 'activate-email' AND iso_639_1 = $1`, h.DbSchema)
	_, err = h.DbPool.Exec(gc, messageQuery, lang)
	if err != nil {
		apiRequest.Error(http.StatusInternalServerError, "failed to create user")
		return
	}
	var subject string
	var template string
	err = h.DbPool.QueryRow(gc, messageQuery, lang).Scan(&subject, &template)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			apiRequest.Error(http.StatusNotFound, "email template not found")
		} else {
			apiRequest.Error(http.StatusInternalServerError, "failed to get email template")
		}
		return
	}

	signupUrl := gc.Request.Referer() + "app/activate/account?token=" + signupTokenString
	emailMessage := strings.Replace(template, "{{link}}", signupUrl, -1)
	emailMessage = strings.Replace(emailMessage, "{{expiry_hours}}", strconv.Itoa(expiryHour), -1)
	go func() {
		sendEmailErr := sendEmail(userCredentials.Email, subject, emailMessage)
		if sendEmailErr != nil {
			apiRequest.Error(http.StatusInternalServerError, "Unable to send email.")
		}
	}()

	apiRequest.SetMeta("user_uuid", userUuid)
	apiRequest.SuccessNoData(http.StatusCreated, "user created successfully")
}

// Permission to use endpoint checked, 2026-01-11, Roald
func (h *ApiHandler) Activate(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "signup")

	var reqeustData struct {
		Token string `json:"token"`
	}
	if err := gc.BindJSON(&reqeustData); err != nil || reqeustData.Token == "" {
		apiRequest.Error(http.StatusBadRequest, "token required")
		return
	}

	// Parse JWT token using the same signing method
	token, err := jwt.ParseWithClaims(reqeustData.Token, &app.Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(h.Config.JwtSecret), nil
	})
	if err != nil {
		apiRequest.Error(http.StatusUnauthorized, "invalid or expired token")
		return
	}

	// Extract claims
	claims, ok := token.Claims.(*app.Claims)
	if !ok || !token.Valid {
		apiRequest.Error(http.StatusUnauthorized, "invalid token")
		return
	}

	userUuid := claims.UserUuid

	// Query stored activation token
	var storedToken string
	query := fmt.Sprintf(`SELECT activate_token FROM %s.user WHERE uuid = $1`, h.DbSchema)
	err = h.DbPool.QueryRow(gc, query, userUuid).Scan(&storedToken)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			apiRequest.Error(http.StatusNotFound, "user not found")
		} else {
			apiRequest.Error(http.StatusInternalServerError, "database error")
		}
		return
	}

	// Compare tokens
	if storedToken != reqeustData.Token {
		apiRequest.Error(http.StatusUnauthorized, "token mismatch")
		return
	}

	// Activate account
	updateQuery := fmt.Sprintf(`UPDATE %s.user SET is_active = true, activate_token = NULL WHERE uuid = $1`, h.DbSchema)
	if _, err := h.DbPool.Exec(gc, updateQuery, userUuid); err != nil {
		apiRequest.Error(http.StatusInternalServerError, "failed to activate user")
		return
	}

	apiRequest.SuccessNoData(http.StatusOK, "account successfully activated")
}
