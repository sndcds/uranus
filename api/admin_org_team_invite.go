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
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
)

type OrganizationTeamInviteClaims struct {
	UserUuid string `json:"user_uuid"`
	OrgUuid  string `json:"org_uuid"`
	jwt.RegisteredClaims
}

type OrganizationTeamInviteInfo struct {
	OrgUuid    string  `json:"org_uuid"`
	OrgName    string  `json:"org_name"`
	OrgCity    *string `json:"org_city,omitempty"`
	OrgCountry *string `json:"org_country,omitempty"`
	OrgWebLink *string `json:"org_web_link,omitempty"`
	OrgEmail   *string `json:"org_email,omitempty"`
}

func (h *ApiHandler) AdminOrganizationTeamInvite(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-org-team-invite")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	lang := gc.DefaultQuery("lang", "en")

	orgUuid := gc.Param("orgUuid")
	if orgUuid == "" {
		apiRequest.Required("orgUuid is required")
		return
	}

	var payload struct {
		Email string `json:"email" binding:"required,email"`
	}

	if err := gc.ShouldBindJSON(&payload); err != nil {
		apiRequest.PayloadError()
		return
	}

	apiMessage := ""

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		var invitedUserUuid string
		var invitedUserDisplayName *string
		var invitedUserFirstName *string
		var invitedUserLastName *string
		var orgName string

		err := tx.QueryRow(
			ctx,
			app.UranusInstance.SqlAdminInvitedOrgTeamMember,
			orgUuid,
			payload.Email).
			Scan(
				&invitedUserUuid,
				&invitedUserDisplayName,
				&invitedUserFirstName,
				&invitedUserLastName,
				&orgName)
		if err != nil {
			if err == pgx.ErrNoRows {
				apiMessage = "user not found"
				return &ApiTxError{
					Code: http.StatusNotFound,
					Err:  errors.New("user not found"),
				}
			}
		}

		// Generate token and send email to user
		expiryMinutes := app.UranusInstance.Config.InvitationExpirationMinutes
		tokenExp := time.Now().Add(time.Duration(expiryMinutes) * time.Minute)
		tokenClaims := &OrganizationTeamInviteClaims{
			UserUuid: invitedUserUuid,
			OrgUuid:  orgUuid,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(tokenExp),
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, tokenClaims)
		tokenString, err := token.SignedString([]byte(h.Config.JwtSecret))
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  errors.New("error signing token"),
			}
		}

		var subject string
		var template string
		err = tx.QueryRow(
			ctx,
			app.UranusInstance.SqlGetSystemEmailTemplate,
			"team-invite-email",
			lang).
			Scan(&subject, &template)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  errors.New("failed to get message template"),
			}
		}

		_, err = tx.Exec(
			ctx,
			app.UranusInstance.SqlAdminInsertInvitedOrgTeamMember,
			orgUuid,
			invitedUserUuid,
			tokenString,
			userUuid)
		if err != nil {
			debugf(err.Error())
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				apiMessage = "user is already invited"
				return &ApiTxError{
					Code: http.StatusConflict,
					Err:  errors.New("user is already invited"),
				}
			}
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  err,
			}
		}

		inviteAcceptUrl := gc.Request.Referer() + "admin/invite/accept?token=" + tokenString
		emailMessage := strings.Replace(template, "{{invite_link}}", inviteAcceptUrl, -1)
		emailMessage = strings.Replace(emailMessage, "{{expiry_minutes}}", strconv.Itoa(expiryMinutes), -1)
		emailMessage = strings.Replace(emailMessage, "{{display_name}}", "Uranus User", -1)
		emailMessage = strings.Replace(emailMessage, "{{organization_name}}", orgName, -1)

		err = sendEmailWithTimeout(payload.Email, subject, emailMessage, 20*time.Second)
		if err != nil {
			debugf(err.Error())
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  err,
			}
		}

		return nil
	})
	if txErr != nil {
		if apiMessage != "" {
			apiRequest.SuccessNoData(http.StatusOK, apiMessage)
			return
		}
		debugf(txErr.Error())
		apiRequest.InternalServerError()
		return
	}

	apiRequest.SuccessNoData(http.StatusCreated, "member invitation sent successfully")
}

func (h *ApiHandler) AdminOrganizationTeamInviteAccept(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-org-team-invite-accept")
	ctx := gc.Request.Context()

	debugf("AdminOrganizationTeamInviteAccept 1")
	var req struct {
		Token string `json:"token"`
	}

	if err := gc.BindJSON(&req); err != nil || req.Token == "" {
		debugf("err 1: %v", err)
		apiRequest.InvalidJSONInput()
		return
	}

	// Parse JWT token
	token, err := jwt.ParseWithClaims(req.Token, &OrganizationTeamInviteClaims{}, func(token *jwt.Token) (interface{}, error) {
		_, ok := token.Method.(*jwt.SigningMethodHMAC)
		if !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(h.Config.JwtSecret), nil
	})
	if err != nil {
		debugf("err 2: %v", err)
		apiRequest.Error(http.StatusUnauthorized, "")
		return
	}

	claims, ok := token.Claims.(*OrganizationTeamInviteClaims)
	if !ok || !token.Valid {
		apiRequest.Error(http.StatusUnauthorized, "")
		return
	}

	userUuid := claims.UserUuid
	orgUuid := claims.OrgUuid

	var orgInfo OrganizationTeamInviteInfo
	orgInfo.OrgUuid = orgUuid

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {

		// Query stored activation token
		var storedToken string
		query := fmt.Sprintf(`SELECT accept_token FROM %s.organization_member_link WHERE user_uuid = $1::uuid AND org_uuid = $2::uuid FOR UPDATE`, h.DbSchema)
		err = tx.QueryRow(ctx, query, userUuid, orgUuid).Scan(&storedToken)
		if err != nil {
			debugf("err 3: %v", err)
			if errors.Is(err, pgx.ErrNoRows) {
				return &ApiTxError{
					Code: http.StatusNotFound,
					Err:  err,
				}
			}
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  err,
			}
		}

		// Compare tokens
		if storedToken != req.Token {
			return &ApiTxError{
				Code: http.StatusUnauthorized,
				Err:  fmt.Errorf("token mismatch"),
			}
		}

		// Activate account
		updateQuery := fmt.Sprintf(
			`UPDATE %s.organization_member_link SET has_joined = true, accept_token = NULL
			WHERE org_uuid = $1::uuid AND user_uuid = $2::uuid`,
			h.DbSchema)
		_, err = tx.Exec(ctx, updateQuery, orgUuid, userUuid)
		if err != nil {
			debugf("err 4: %v", err)
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("failed to accept invite"),
			}
		}

		// Create user organization link
		uolQuery := fmt.Sprintf(`
			INSERT INTO %s.user_organization_link (user_uuid, org_uuid, permissions)
			VALUES ($1::uuid, $2::uuid, $3)`,
			h.DbSchema)
		_, err = tx.Exec(ctx, uolQuery, userUuid, orgUuid, 0)
		if err != nil {
			debugf("err 5: %v", err)
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("failed to accept invite"),
			}
		}

		orgQuery := fmt.Sprintf(`
			SELECT name, city, country, web_link, contact_email FROM %s.organization WHERE uuid = $1::uuid`,
			h.DbSchema)
		err = tx.QueryRow(ctx, orgQuery, orgUuid).
			Scan(&orgInfo.OrgName, &orgInfo.OrgCity, &orgInfo.OrgCountry, &orgInfo.OrgWebLink, &orgInfo.OrgEmail)
		if err != nil {
			debugf("err 6: %v", err)
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("failed to get organization info"),
			}
		}

		return nil
	})
	if txErr != nil {
		debugf(txErr.Error())
		apiRequest.InternalServerError()
		return
	}

	apiRequest.Success(http.StatusOK, orgInfo, "user joined successfully")
}
