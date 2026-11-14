package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
)

type OrgTeamInviteClaims struct {
	UserUuid string `json:"user_uuid"`
	OrgUuid  string `json:"org_uuid"`
	jwt.RegisteredClaims
}

type OrgTeamInviteInfo struct {
	OrgUuid    string  `json:"org_uuid"`
	OrgName    string  `json:"org_name"`
	OrgCity    *string `json:"org_city,omitempty"`
	OrgCountry *string `json:"org_country,omitempty"`
	OrgWebLink *string `json:"org_web_link,omitempty"`
	OrgEmail   *string `json:"org_email,omitempty"`
}

type OrgTeamInvitedUser struct {
	Uuid        string
	Email       string
	Locale      string
	DisplayName *string
	FirstName   *string
	LastName    *string
	OrgName     *string
	AcceptUrl   string
}

//------------------------------------------------------------------------------
// 	AdminOrgTeamInvite
//------------------------------------------------------------------------------

func (h *ApiHandler) AdminOrgTeamInvite(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-org-team-invite")

	ctx, cancel := context.WithTimeout(gc.Request.Context(), 90*time.Second)
	defer cancel()

	userUuid := h.userUuid(gc)
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

	expiryMinutes := app.UranusInstance.Config.InvitationExpirationMinutes
	if expiryMinutes <= 0 {
		debugf("invalid invitation expiration: %d minutes", expiryMinutes)
		apiRequest.InternalServerError()
		return
	}

	frontendDashboard := strings.TrimRight(h.Config.FrontendDashboard, "/")
	if frontendDashboard == "" {
		debugf("frontend dashboard URL is not configured")
		apiRequest.InternalServerError()
		return
	}

	var invitedUser OrgTeamInvitedUser
	var emailSubject string
	var emailMessage string
	var userAlreadyJoined bool

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {

		//----------------------------------------------------------------------
		// 	Check permissions
		//----------------------------------------------------------------------

		if txErr := h.CheckOrgPermissionTx(
			gc,
			tx,
			userUuid,
			orgUuid,
			app.UserPermManageTeam,
		); txErr != nil {
			return txErr
		}

		//----------------------------------------------------------------------
		// 	Fetch invited user and organization information
		//----------------------------------------------------------------------

		err := tx.QueryRow(
			ctx,
			app.UranusInstance.SqlAdminInvitedOrgTeamMember,
			orgUuid,
			payload.Email,
		).Scan(
			&invitedUser.Uuid,
			&invitedUser.DisplayName,
			&invitedUser.FirstName,
			&invitedUser.LastName,
			&invitedUser.OrgName,
			&invitedUser.Locale,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return &ApiTxError{
					Code: http.StatusNotFound,
					Err:  errors.New("user not found"),
				}
			}

			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("failed to fetch invited user: %w", err),
			}
		}

		invitedUser.Email = payload.Email

		//----------------------------------------------------------------------
		//  Generate invitation token
		//----------------------------------------------------------------------

		tokenExp := time.Now().Add(
			time.Duration(expiryMinutes) * time.Minute,
		)

		tokenClaims := &OrgTeamInviteClaims{
			UserUuid: invitedUser.Uuid,
			OrgUuid:  orgUuid,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(tokenExp),
			},
		}

		token := jwt.NewWithClaims(
			jwt.SigningMethodHS256,
			tokenClaims,
		)

		tokenString, err := token.SignedString([]byte(h.Config.JwtSecret))
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("failed to sign invitation token: %w", err),
			}
		}

		//----------------------------------------------------------------------
		//  Create or update invitation
		//----------------------------------------------------------------------

		res, err := tx.Exec(
			ctx,
			app.UranusInstance.SqlAdminUpsertInvitedOrgTeamMember,
			orgUuid,
			invitedUser.Uuid,
			tokenString,
			userUuid,
		)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("failed to create invitation: %w", err),
			}
		}

		if res.RowsAffected() == 0 {
			userAlreadyJoined = true
			return nil
		}

		//----------------------------------------------------------------------
		//  Prepare invitation URL
		//----------------------------------------------------------------------

		invitedUser.AcceptUrl = fmt.Sprintf(
			"%s/app/activate/team-invitation?token=%s",
			frontendDashboard,
			url.QueryEscape(tokenString),
		)

		//----------------------------------------------------------------------
		//  Render invitation email
		//
		//  Rendering happens before commit. A rendering error therefore
		//  rolls back the invitation. Sending happens after commit.
		//----------------------------------------------------------------------

		locale := app.NormalizeLocale(invitedUser.Locale)

		layoutPath := fmt.Sprintf(
			"template/email/layout/%s.html",
			locale,
		)

		contentPath := fmt.Sprintf(
			"template/email/team-invite/%s.html",
			locale,
		)

		displayName := BuildUserLabel(
			invitedUser.Email,
			invitedUser.DisplayName,
			invitedUser.FirstName,
			invitedUser.LastName,
		)

		data := struct {
			Language         string
			DisplayName      string
			OrganizationName *string
			InviteLink       string
			ExpiryMinutes    int
		}{
			Language:         locale,
			DisplayName:      displayName,
			OrganizationName: invitedUser.OrgName,
			InviteLink:       invitedUser.AcceptUrl,
			ExpiryMinutes:    expiryMinutes,
		}

		emailSubject, emailMessage, err = app.RenderEmailTemplate(
			layoutPath,
			contentPath,
			data,
		)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("failed to render invitation email: %w", err),
			}
		}

		return nil
	})

	if txErr != nil {
		debugf(txErr.Error())
		apiRequest.Error(
			txErr.Code,
			"invitation could not be created",
		)
		return
	}

	if userAlreadyJoined {
		apiRequest.SuccessNoData(
			http.StatusOK,
			"user has already joined the organization",
		)
		return
	}

	//--------------------------------------------------------------------------
	//  Send email asynchronously, after the transaction has committed
	//--------------------------------------------------------------------------

	go func() {
		if err := app.SendEmailWithTimeout(
			invitedUser.Email,
			emailSubject,
			emailMessage,
			20*time.Second,
		); err != nil {
			debugf("team invitation email failed: %v", err)
		}
	}()

	apiRequest.SuccessNoData(
		http.StatusCreated,
		"invitation created successfully",
	)
}

//------------------------------------------------------------------------------
//  OrgTeamInviteAccept
//------------------------------------------------------------------------------

func (h *ApiHandler) OrgTeamInviteAccept(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "org-team-invite-accept")

	ctx, cancel := context.WithTimeout(gc.Request.Context(), 90*time.Second)
	defer cancel()

	var req struct {
		Token string `json:"token" binding:"required"`
	}

	if err := gc.ShouldBindJSON(&req); err != nil {
		apiRequest.InvalidJSONInput()
		return
	}

	//--------------------------------------------------------------------------
	//  Parse and validate invitation token
	//--------------------------------------------------------------------------

	token, err := jwt.ParseWithClaims(
		req.Token,
		&OrgTeamInviteClaims{},
		func(token *jwt.Token) (interface{}, error) {
			return []byte(h.Config.JwtSecret), nil
		},
		jwt.WithValidMethods([]string{
			jwt.SigningMethodHS256.Alg(),
		}),
		jwt.WithExpirationRequired(),
	)
	if err != nil || token == nil {
		apiRequest.Error(
			http.StatusUnauthorized,
			"invalid invitation",
		)
		return
	}

	claims, ok := token.Claims.(*OrgTeamInviteClaims)
	if !ok || claims == nil || !token.Valid || claims.ExpiresAt == nil {
		apiRequest.Error(
			http.StatusUnauthorized,
			"invalid invitation",
		)
		return
	}

	userUuid := claims.UserUuid
	orgUuid := claims.OrgUuid

	if _, err := uuid.Parse(userUuid); err != nil {
		apiRequest.Error(http.StatusUnauthorized, "invalid invitation")
		return
	}

	if _, err := uuid.Parse(orgUuid); err != nil {
		apiRequest.Error(http.StatusUnauthorized, "invalid invitation")
		return
	}

	var orgInfo OrgTeamInviteInfo
	orgInfo.OrgUuid = orgUuid

	var invitedByUserUuid *string

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {

		//----------------------------------------------------------------------
		//  Load and lock invitation
		//----------------------------------------------------------------------

		query := fmt.Sprintf(`
			SELECT
				accept_token,
				invited_by_user_uuid,
				has_joined
			FROM %s.organization_member_link
			WHERE user_uuid = $1::uuid
			  AND org_uuid = $2::uuid
			FOR UPDATE`,
			h.DbSchema,
		)

		var storedToken *string
		var hasJoined bool

		err := tx.QueryRow(
			ctx,
			query,
			userUuid,
			orgUuid,
		).Scan(
			&storedToken,
			&invitedByUserUuid,
			&hasJoined,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return &ApiTxError{
					Code: http.StatusNotFound,
					Err:  errors.New("invitation not found"),
				}
			}

			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("failed to load invitation: %w", err),
			}
		}

		//----------------------------------------------------------------------
		//  Validate stored invitation
		//----------------------------------------------------------------------

		if hasJoined {
			return &ApiTxError{
				Code: http.StatusConflict,
				Err:  errors.New("invitation has already been accepted"),
			}
		}

		if storedToken == nil || *storedToken == "" {
			return &ApiTxError{
				Code: http.StatusUnauthorized,
				Err:  errors.New("invalid invitation"),
			}
		}

		if *storedToken != req.Token {
			return &ApiTxError{
				Code: http.StatusUnauthorized,
				Err:  errors.New("invalid invitation"),
			}
		}

		//----------------------------------------------------------------------
		//  Accept invitation
		//----------------------------------------------------------------------

		updateQuery := fmt.Sprintf(`
			UPDATE %s.organization_member_link
			SET
				has_joined = TRUE,
				accept_token = NULL
			WHERE org_uuid = $1::uuid
			  AND user_uuid = $2::uuid`,
			h.DbSchema,
		)

		res, err := tx.Exec(
			ctx,
			updateQuery,
			orgUuid,
			userUuid,
		)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("failed to accept invitation: %w", err),
			}
		}

		if res.RowsAffected() != 1 {
			return &ApiTxError{
				Code: http.StatusConflict,
				Err:  errors.New("invitation could not be accepted"),
			}
		}

		//----------------------------------------------------------------------
		//  Create user/organization link
		//----------------------------------------------------------------------

		uolQuery := fmt.Sprintf(`
			INSERT INTO %s.user_organization_link (
				user_uuid,
				org_uuid,
				permissions
			)
			VALUES ($1::uuid, $2::uuid, $3)
			ON CONFLICT (user_uuid, org_uuid) DO NOTHING`,
			h.DbSchema,
		)

		_, err = tx.Exec(
			ctx,
			uolQuery,
			userUuid,
			orgUuid,
			0,
		)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("failed to add user to organization: %w", err),
			}
		}

		//----------------------------------------------------------------------
		//  Fetch organization information
		//----------------------------------------------------------------------

		orgQuery := fmt.Sprintf(`
			SELECT
				name,
				city,
				country,
				web_link,
				contact_email
			FROM %s.organization
			WHERE uuid = $1::uuid`,
			h.DbSchema,
		)

		err = tx.QueryRow(
			ctx,
			orgQuery,
			orgUuid,
		).Scan(
			&orgInfo.OrgName,
			&orgInfo.OrgCity,
			&orgInfo.OrgCountry,
			&orgInfo.OrgWebLink,
			&orgInfo.OrgEmail,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return &ApiTxError{
					Code: http.StatusNotFound,
					Err:  errors.New("organization not found"),
				}
			}

			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("failed to get organization information: %w", err),
			}
		}

		return nil
	})

	if txErr != nil {
		debugf(txErr.Error())
		apiRequest.Error(txErr.Code, txErr.Error())
		return
	}

	//--------------------------------------------------------------------------
	//  Notify inviter after successful commit
	//--------------------------------------------------------------------------

	if invitedByUserUuid != nil {
		go func() {
			if err := h.sendOrgInviteAcceptedEmail(
				userUuid,
				*invitedByUserUuid,
				orgUuid,
			); err != nil {
				debugf("invite acceptance email failed: %v", err)
			}
		}()
	}

	apiRequest.Success(
		http.StatusOK,
		orgInfo,
		"user joined successfully",
	)
}

//------------------------------------------------------------------------------
//  sendOrgInviteAcceptedEmail
//------------------------------------------------------------------------------

func (h *ApiHandler) sendOrgInviteAcceptedEmail(
	userUuid string,
	invitedByUserUuid string,
	orgUuid string,
) error {

	ctx, cancel := context.WithTimeout(
		context.Background(),
		90*time.Second,
	)
	defer cancel()

	//--------------------------------------------------------------------------
	//  Fetch email information
	//--------------------------------------------------------------------------

	userEmail, err := h.GetUserEmail(ctx, userUuid)
	if err != nil {
		return fmt.Errorf("failed to get accepted user's email: %w", err)
	}

	inviterEmail, err := h.GetUserEmail(ctx, invitedByUserUuid)
	if err != nil {
		return fmt.Errorf("failed to get inviter's email: %w", err)
	}

	orgName, orgCity, err := h.GetOrgNameAndCity(ctx, orgUuid)
	if err != nil {
		return fmt.Errorf("failed to get organization information: %w", err)
	}

	locale, err := h.GetUserLocale(ctx, invitedByUserUuid)
	if err != nil {
		return fmt.Errorf("failed to get inviter's locale: %w", err)
	}

	locale = app.NormalizeLocale(locale)

	//--------------------------------------------------------------------------
	//  Render email
	//--------------------------------------------------------------------------

	layoutPath := fmt.Sprintf(
		"template/email/layout/%s.html",
		locale,
	)

	contentPath := fmt.Sprintf(
		"template/email/team-member-accepted/%s.html",
		locale,
	)

	frontendDashboard := strings.TrimRight(h.Config.FrontendDashboard, "/")
	if frontendDashboard == "" {
		return errors.New("frontend dashboard URL is not configured")
	}

	redirectLink := fmt.Sprintf(
		"%s/admin/org/%s/member/%s/permissions",
		frontendDashboard,
		orgUuid,
		userUuid,
	)

	data := struct {
		Language     string
		UserEmail    string
		OrgName      string
		OrgCity      string
		RedirectLink string
	}{
		Language:     locale,
		UserEmail:    userEmail,
		OrgName:      orgName,
		OrgCity:      orgCity,
		RedirectLink: redirectLink,
	}

	subject, body, err := app.RenderEmailTemplate(
		layoutPath,
		contentPath,
		data,
	)
	if err != nil {
		return fmt.Errorf("failed to render invitation acceptance email: %w", err)
	}

	//--------------------------------------------------------------------------
	//  Send email
	//--------------------------------------------------------------------------

	return app.SendEmailWithTimeout(
		inviterEmail,
		subject,
		body,
		60*time.Second,
	)
}
