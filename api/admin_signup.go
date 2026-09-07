package api

import (
	"context"
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
	"github.com/sndcds/grains/grains_validation"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) Signup(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "signup")
	ctx := gc.Request.Context()
	lang := gc.DefaultQuery("lang", "en")

	var payload struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
		Referer  string `json:"referer" binding:"required"`
	}

	if err := gc.ShouldBindJSON(&payload); err != nil {
		apiRequest.PayloadError()
		return
	}

	// Validate the password before performing the expensive bcrypt hash.
	if err := grains_validation.ValidatePassword(
		payload.Email,
		payload.Password,
		grains_validation.DefaultMinPasswordLength,
	); err != nil {
		debugf("invalid password during signup: %v", err)

		// Do not expose the exact validation reason to the client.
		apiRequest.Error(
			http.StatusUnprocessableEntity,
			"(#2) password does not meet security requirements",
		)
		return
	}

	if !app.IsValidEmail(payload.Email) {
		apiRequest.Error(
			http.StatusBadRequest,
			"(#3) invalid email",
		)
		return
	}

	passwordHash, err := app.EncryptPassword(payload.Password)
	if err != nil {
		debugf("failed to hash password: %v", err)
		apiRequest.InternalServerError()
		return
	}

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		// Check if user already exists.
		var exists bool

		checkQuery := fmt.Sprintf(
			"SELECT EXISTS(SELECT 1 FROM %s.user WHERE email = $1)",
			h.DbSchema,
		)

		err := tx.QueryRow(
			ctx,
			checkQuery,
			payload.Email,
		).Scan(&exists)

		if err != nil {
			return TxInternalError(nil)
		}

		if exists {
			return TxInternalError(nil)
		}

		userUuid, err := grains_uuid.Uuidv7String()
		if err != nil {
			return TxInternalError(nil)
		}

		insertQuery := fmt.Sprintf(
			`INSERT INTO %s.user
				(uuid, email, password_hash)
			 VALUES
				($1::uuid, $2, $3)`,
			h.DbSchema,
		)

		_, err = tx.Exec(
			ctx,
			insertQuery,
			userUuid,
			payload.Email,
			passwordHash,
		)

		if err != nil {
			return TxInternalError(nil)
		}

		// Generate account activation token.
		const expiryHours = 1

		signupExp := time.Now().Add(
			time.Duration(expiryHours) * time.Hour,
		)

		signupClaims := &app.Claims{
			UserUuid: userUuid,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(signupExp),
			},
		}

		signupToken := jwt.NewWithClaims(
			jwt.SigningMethodHS256,
			signupClaims,
		)

		signupTokenString, err := signupToken.SignedString(
			[]byte(h.Config.JwtSecret),
		)
		if err != nil {
			return TxInternalError(nil)
		}

		updateQuery := fmt.Sprintf(
			`UPDATE %s.user
			 SET activate_token = $1
			 WHERE uuid = $2::uuid`,
			h.DbSchema,
		)

		_, err = tx.Exec(
			ctx,
			updateQuery,
			signupTokenString,
			userUuid,
		)

		if err != nil {
			return TxInternalError(nil)
		}

		// Load email template.
		messageQuery := fmt.Sprintf(
			`SELECT subject, template
			 FROM %s.system_email_template
			 WHERE context = 'user-email-verification'
			   AND iso_639_1 = $1`,
			h.DbSchema,
		)

		var subject string
		var template string

		err = tx.QueryRow(
			ctx,
			messageQuery,
			lang,
		).Scan(&subject, &template)

		if err != nil {
			return TxInternalError(nil)
		}

		signupURL := payload.Referer +
			"/app/activate/account?token=" +
			signupTokenString

		emailMessage := strings.ReplaceAll(
			template,
			"{{link}}",
			signupURL,
		)

		emailMessage = strings.ReplaceAll(
			emailMessage,
			"{{expiry_hours}}",
			strconv.Itoa(expiryHours),
		)

		if err := sendEmailWithTimeout(
			payload.Email,
			subject,
			emailMessage,
			20*time.Second,
		); err != nil {
			return TxInternalError(nil)
		}

		apiRequest.SetMeta("user_uuid", userUuid)

		return nil
	})

	if txErr != nil {
		debugf(txErr.Error())
		apiRequest.Error(txErr.Code, txErr.Error())
		return
	}

	apiRequest.SuccessNoData(
		http.StatusCreated,
		"user registered successfully",
	)
}

func sendEmailWithContext(ctx context.Context, to, subject, body string) error {
	done := make(chan error, 1)
	go func() {
		done <- sendEmail(to, subject, body) // your existing sendEmail
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (h *ApiHandler) Activate(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "signup")

	var requestData struct {
		Token string `json:"token"`
	}
	if err := gc.BindJSON(&requestData); err != nil || requestData.Token == "" {
		apiRequest.Required("token required")
		return
	}

	// Parse JWT token using the same signing method
	token, err := jwt.ParseWithClaims(
		requestData.Token,
		&app.Claims{},
		func(token *jwt.Token) (interface{}, error) {
			return []byte(h.Config.JwtSecret), nil
		},
		jwt.WithValidMethods([]string{
			jwt.SigningMethodHS256.Alg(),
		}),
	)
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
	query := fmt.Sprintf(`SELECT activate_token FROM %s.user WHERE uuid = $1::uuid`, h.DbSchema)
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
	if storedToken != requestData.Token {
		apiRequest.Error(http.StatusUnauthorized, "token mismatch")
		return
	}

	// Activate account
	updateQuery := fmt.Sprintf(`UPDATE %s.user SET is_active = true, activate_token = NULL WHERE uuid = $1::uuid`, h.DbSchema)
	if _, err := h.DbPool.Exec(gc, updateQuery, userUuid); err != nil {
		apiRequest.Error(http.StatusInternalServerError, "failed to activate user")
		return
	}

	apiRequest.SuccessNoData(http.StatusOK, "account successfully activated")
}
