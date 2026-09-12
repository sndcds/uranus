package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
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

type OrgTeamInviteXX struct {
	Uuid        string
	Email       string
	DisplayName *string
	FirstName   *string
	LastName    *string
	OrgName     *string
	AcceptUrl   string
}

func (h *ApiHandler) AdminOrgTeamInvite(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-org-team-invite")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	orgUuid := gc.Param("orgUuid")
	if orgUuid == "" {
		apiRequest.Required("orgUuid is required")
		return
	}

	var payload struct {
		Email   string `json:"email" binding:"required,email"`
		Referer string `json:"referer" binding:"required"`
	}

	if err := gc.ShouldBindJSON(&payload); err != nil {
		apiRequest.PayloadError()
		return
	}

	var invitedUser OrgTeamInviteXX
	var userAlreadyJoined bool

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {

		// ---------------------------------------------------------------------
		// Check permissions
		// ---------------------------------------------------------------------

		txErr := h.CheckOrgPermissionTx(
			gc,
			tx,
			userUuid,
			orgUuid,
			app.UserPermManageTeam,
		)
		if txErr != nil {
			return txErr
		}

		// ---------------------------------------------------------------------
		// Fetch invited user and organization information
		// ---------------------------------------------------------------------

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
				Err:  err,
			}
		}

		invitedUser.Email = payload.Email

		// ---------------------------------------------------------------------
		// Generate invitation token
		// ---------------------------------------------------------------------

		expiryMinutes := app.UranusInstance.Config.InvitationExpirationMinutes

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

		tokenString, err := token.SignedString(
			[]byte(h.Config.JwtSecret),
		)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  errors.New("error signing invitation token"),
			}
		}

		// ---------------------------------------------------------------------
		// Create/update invitation
		// ---------------------------------------------------------------------

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
				Err:  err,
			}
		}

		if res.RowsAffected() == 0 {
			userAlreadyJoined = true
			return nil
		}

		// ---------------------------------------------------------------------
		// Prepare invitation URL
		//
		// The URL is only prepared here. The email is rendered and sent after
		// the transaction has successfully committed.
		// ---------------------------------------------------------------------

		invitedUser.AcceptUrl =
			payload.Referer +
				"/app/activate/team-invitation?token=" +
				tokenString

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

	// The user is already a member, so no invitation email is required.
	if userAlreadyJoined {
		apiRequest.SuccessNoData(
			http.StatusCreated,
			"user has already joined the organization",
		)
		return
	}

	// -------------------------------------------------------------------------
	// Render invitation email
	//
	// The transaction has already committed at this point.
	// -------------------------------------------------------------------------

	locale, err := h.GetUserLocale(ctx, invitedUser.Uuid)
	if err != nil {
		debugf("failed to get invited user's locale: %v", err)
		apiRequest.InternalServerError()
		return
	}

	locale = app.NormalizeLocale(locale)

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
		ExpiryMinutes:    app.UranusInstance.Config.InvitationExpirationMinutes,
	}

	subject, emailMessage, err := app.RenderEmailTemplate(
		layoutPath,
		contentPath,
		data,
	)
	if err != nil {
		debugf("failed to render team invitation email: %v", err)
		apiRequest.InternalServerError()
		return
	}

	// -------------------------------------------------------------------------
	// Send email asynchronously
	// -------------------------------------------------------------------------

	go func() {
		if err := app.SendEmailWithTimeout(
			payload.Email,
			subject,
			emailMessage,
			20*time.Second,
		); err != nil {
			debugf("team invitation email failed: %v", err)
		}
	}()

	apiRequest.SuccessNoData(
		http.StatusCreated,
		"member invitation sent successfully",
	)
}

func (h *ApiHandler) OrgTeamInviteAccept(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "org-team-invite-accept")
	ctx := gc.Request.Context()

	var req struct {
		Token string `json:"token"`
	}

	if err := gc.BindJSON(&req); err != nil || req.Token == "" {
		apiRequest.InvalidJSONInput()
		return
	}

	// -------------------------------------------------------------------------
	// Parse and validate invitation token
	// -------------------------------------------------------------------------

	token, err := jwt.ParseWithClaims(
		req.Token,
		&OrgTeamInviteClaims{},
		func(token *jwt.Token) (interface{}, error) {
			return []byte(h.Config.JwtSecret), nil
		},
		jwt.WithValidMethods([]string{
			jwt.SigningMethodHS256.Alg(),
		}),
	)
	if err != nil {
		apiRequest.Error(
			http.StatusUnauthorized,
			"invalid invitation",
		)
		return
	}

	claims, ok := token.Claims.(*OrgTeamInviteClaims)
	if !ok || !token.Valid || claims.ExpiresAt == nil {
		apiRequest.Error(
			http.StatusUnauthorized,
			"invalid invitation",
		)
		return
	}

	userUuid := claims.UserUuid
	orgUuid := claims.OrgUuid

	if _, err := uuid.Parse(userUuid); err != nil {
		apiRequest.Error(
			http.StatusUnauthorized,
			"invalid invitation",
		)
		return
	}

	if _, err := uuid.Parse(orgUuid); err != nil {
		apiRequest.Error(
			http.StatusUnauthorized,
			"invalid invitation",
		)
		return
	}

	var orgInfo OrgTeamInviteInfo
	orgInfo.OrgUuid = orgUuid

	var invitedByUserUuid *string

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {

		// ---------------------------------------------------------------------
		// Load and lock invitation
		// ---------------------------------------------------------------------

		var storedToken *string
		var hasJoined bool

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
				Err:  err,
			}
		}

		// ---------------------------------------------------------------------
		// Validate stored invitation
		// ---------------------------------------------------------------------

		if hasJoined {
			return &ApiTxError{
				Code: http.StatusUnauthorized,
				Err:  errors.New("already joined"),
			}
		}

		if storedToken == nil || *storedToken == "" {
			return &ApiTxError{
				Code: http.StatusUnauthorized,
				Err:  errors.New("missing invitation token"),
			}
		}

		if *storedToken != req.Token {
			return &ApiTxError{
				Code: http.StatusUnauthorized,
				Err:  errors.New("invitation token mismatch"),
			}
		}

		// ---------------------------------------------------------------------
		// Accept invitation
		// ---------------------------------------------------------------------

		updateQuery := fmt.Sprintf(
			`UPDATE %s.organization_member_link
			 SET has_joined = TRUE,
			     accept_token = NULL
			 WHERE org_uuid = $1::uuid
			   AND user_uuid = $2::uuid`,
			h.DbSchema,
		)

		_, err = tx.Exec(
			ctx,
			updateQuery,
			orgUuid,
			userUuid,
		)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  errors.New("failed to accept invitation"),
			}
		}

		// ---------------------------------------------------------------------
		// Create user/organization link
		// ---------------------------------------------------------------------

		uolQuery := fmt.Sprintf(`
			INSERT INTO %s.user_organization_link (user_uuid, org_uuid, permissions)
			VALUES ($1::uuid, $2::uuid, $3)
			ON CONFLICT (user_uuid, org_uuid) DO NOTHING`,
			h.DbSchema)

		_, err = tx.Exec(ctx, uolQuery, userUuid, orgUuid, 0)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("failed to add user to organization: %w", err),
			}
		}

		// ---------------------------------------------------------------------
		// Fetch organization information
		// ---------------------------------------------------------------------

		orgQuery := fmt.Sprintf(
			`SELECT
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
				Err:  errors.New("failed to get organization information"),
			}
		}

		return nil
	})

	if txErr != nil {
		debugf(txErr.Error())
		apiRequest.Error(txErr.Code, txErr.Error())
		return
	}

	// -------------------------------------------------------------------------
	// Notify inviter after successful commit
	// -------------------------------------------------------------------------

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

	apiRequest.Success(http.StatusOK, orgInfo, "user joined successfully")
}

func (h *ApiHandler) sendOrgInviteAcceptedEmail(
	userUuid string,
	invitedByUserUuid string,
	orgUuid string,
) error {

	ctx := context.Background()

	userEmail, err := h.GetUserEmail(ctx, userUuid)
	if err != nil {
		return err
	}

	inviterEmail, err := h.GetUserEmail(ctx, invitedByUserUuid)
	if err != nil {
		return err
	}

	orgName, orgCity, err := h.GetOrgNameAndCity(ctx, orgUuid)
	if err != nil {
		return err
	}

	locale, err := h.GetUserLocale(ctx, invitedByUserUuid)
	if err != nil {
		return err
	}

	locale = app.NormalizeLocale(locale)

	layoutPath := fmt.Sprintf(
		"template/email/layout/%s.html",
		locale,
	)

	contentPath := fmt.Sprintf(
		"template/email/team-member-accepted/%s.html",
		locale,
	)

	// http://localhost:5173/admin/org/{orguuid}/member/{invitedByUserUuid}/permissions

	redirectLink := fmt.Sprintf(
		"%s/admin/org/%s/member/%s/permissions",
		app.UranusInstance.Config.FrontendDashboard,
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
		return err
	}

	return app.SendEmailWithTimeout(
		inviterEmail,
		subject,
		body,
		60*time.Second,
	)
}
