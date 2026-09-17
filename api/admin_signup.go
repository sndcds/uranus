package api

import (
	"errors"
	"fmt"
	"math"
	"net/http"
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

	var payload struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
		Referer  string `json:"referer" binding:"required"`
	}

	if err := gc.ShouldBindJSON(&payload); err != nil {
		apiRequest.PayloadError()
		return
	}

	locale := app.NormalizeLocale(
		gc.DefaultQuery("lang", "en"),
	)

	//--------------------------------------------------------------------------
	// Validate input
	//--------------------------------------------------------------------------

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

	//--------------------------------------------------------------------------
	// Hash password
	//--------------------------------------------------------------------------

	passwordHash, err := app.EncryptPassword(payload.Password)
	if err != nil {
		debugf("failed to hash password: %v", err)
		apiRequest.InternalServerError()
		return
	}

	//--------------------------------------------------------------------------
	// Create user
	//--------------------------------------------------------------------------

	expiryMinutes := app.UranusInstance.Config.SignupTokenExpirationTime

	var userUuid string
	var signupTokenString string

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {

		// Check if user already exists.
		query := fmt.Sprintf(
			"SELECT EXISTS(SELECT 1 FROM %s.user WHERE email = $1)",
			h.DbSchema,
		)

		var exists bool
		if err := tx.QueryRow(
			ctx,
			query,
			payload.Email,
		).Scan(&exists); err != nil {
			return TxInternalError(err)
		}

		if exists {
			return TxInternalError(nil)
		}

		// Generate user UUID.
		userUuid, err = grains_uuid.Uuidv7String()
		if err != nil {
			return TxInternalError(err)
		}

		// Insert user.
		query = fmt.Sprintf(
			`INSERT INTO %s.user
			(uuid, email, password_hash)
			VALUES
			($1::uuid, $2, $3)`,
			h.DbSchema,
		)

		if _, err = tx.Exec(
			ctx,
			query,
			userUuid,
			payload.Email,
			passwordHash,
		); err != nil {
			return TxInternalError(err)
		}

		// Generate account activation token.
		signupExp := time.Now().Add(
			time.Duration(expiryMinutes) * time.Minute,
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

		signupTokenString, err = signupToken.SignedString(
			[]byte(h.Config.JwtSecret),
		)
		if err != nil {
			return TxInternalError(err)
		}

		// Store activation token.
		query = fmt.Sprintf(
			`UPDATE %s.user
			SET activate_token = $1
			WHERE uuid = $2::uuid`,
			h.DbSchema,
		)

		if _, err = tx.Exec(
			ctx,
			query,
			signupTokenString,
			userUuid,
		); err != nil {
			return TxInternalError(err)
		}

		return nil
	})

	if txErr != nil {
		debugf(txErr.Error())
		apiRequest.Error(txErr.Code, txErr.Error())
		return
	}

	//--------------------------------------------------------------------------
	// Render verification email
	//--------------------------------------------------------------------------

	verificationURL := payload.Referer +
		"/app/activate/account?token=" +
		signupTokenString

	layoutPath := fmt.Sprintf(
		"template/email/layout/%s.html",
		locale,
	)

	contentPath := fmt.Sprintf(
		"template/email/user-email-verification/%s.html",
		locale,
	)

	data := struct {
		Language         string
		VerificationLink string
		ExpiryHours      int
	}{
		Language:         locale,
		VerificationLink: verificationURL,
		ExpiryHours:      int(math.Round(float64(expiryMinutes) / 60)),
	}

	subject, emailContent, err := app.RenderEmailTemplate(
		layoutPath,
		contentPath,
		data,
	)
	if err != nil {
		debugf("failed to render email verification email: %v", err)
		apiRequest.InternalServerError()
		return
	}

	//--------------------------------------------------------------------------
	// Send email
	//--------------------------------------------------------------------------

	go func() {
		if err := app.SendEmailWithTimeout(
			payload.Email,
			subject,
			emailContent,
			20*time.Second,
		); err != nil {
			debugf("email verification email failed: %v", err)
		}
	}()

	apiRequest.SetMeta("user_uuid", userUuid)

	apiRequest.SuccessNoData(
		http.StatusCreated,
		"user registered successfully",
	)
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

	claims, err := app.ParseJWT(requestData.Token)
	if err != nil || !app.ValidUUID(claims.UserUuid) {
		apiRequest.Error(http.StatusUnauthorized, "invalid or expired token")
		return
	}

	userUuid := claims.UserUuid

	// Query stored activation token
	var storedToken string

	query := fmt.Sprintf(`
		SELECT activate_token
		FROM %s.user
		WHERE uuid = $1::uuid`,
		h.DbSchema)
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
	updateQuery := fmt.Sprintf(`
		UPDATE %s.user
		SET is_active = true, activate_token = NULL
		WHERE uuid = $1::uuid`,
		h.DbSchema)

	if _, err := h.DbPool.Exec(gc, updateQuery, userUuid); err != nil {
		apiRequest.Error(http.StatusInternalServerError, "failed to activate user")
		return
	}

	apiRequest.SuccessNoData(http.StatusOK, "account successfully activated")
}
