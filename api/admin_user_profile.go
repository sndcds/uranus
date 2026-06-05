package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

func (h *ApiHandler) AdminGetUserProfile(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-get-user-profile")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	query := fmt.Sprintf(`
        SELECT email, username, display_name, first_name, last_name, locale, theme
        FROM %s.user
        WHERE uuid = $1
        `,
		h.DbSchema)

	var profile model.UserProfileResponse
	profile.UserUUID = userUuid
	row := h.DbPool.QueryRow(ctx, query, userUuid)
	err := row.Scan(
		&profile.Email,
		&profile.Username,
		&profile.DisplayName,
		&profile.FirstName,
		&profile.LastName,
		&profile.Locale,
		&profile.Theme)
	if err != nil {
		if err == pgx.ErrNoRows {
			apiRequest.NotFound("user not found")
		} else {
			debugf(err.Error())
			apiRequest.InternalServerError()
		}
		return
	}

	profile.AvatarUrl = app.GetAvatarURL(h.Config.BaseApiUrl, h.Config.ProfileImageDir, userUuid, 64)

	apiRequest.Success(http.StatusOK, profile, "")
}

func (h *ApiHandler) AdminUpdateUserProfile(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-update-user-profile")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	var payload struct {
		Email       string  `json:"email" binding:"required"`
		Username    *string `json:"username"`
		DisplayName *string `json:"display_name"`
		FirstName   *string `json:"first_name"`
		LastName    *string `json:"last_name"`
		Locale      string  `json:"locale" binding:"required"`
		Theme       string  `json:"theme" binding:"required"`
	}

	if err := gc.ShouldBindJSON(&payload); err != nil {
		apiRequest.InvalidJSONInput()
		return
	}

	// TODO: implement full email validation
	if !strings.Contains(payload.Email, "@") {
		apiRequest.Error(http.StatusBadRequest, "Invalid email")
		return
	}

	checkQuery := fmt.Sprintf(`SELECT uuid FROM %s.user WHERE email = $1`, h.DbSchema)

	var existingUserUuid string
	err := h.DbPool.QueryRow(ctx, checkQuery, payload.Email).Scan(&existingUserUuid)

	if err != nil && err != pgx.ErrNoRows {
		debugf(err.Error())
		apiRequest.Error(http.StatusBadRequest, "Failed to check existing email")
		return
	}

	if err == nil && existingUserUuid != userUuid {
		apiRequest.Error(http.StatusConflict, "Email address already exists")
		return
	}

	updateQuery := fmt.Sprintf(`
        UPDATE %s.user
        SET
            email = $1,
            username = $2,
            display_name = $3,
            first_name = $4,
            last_name = $5,
            locale = $6,
            theme = $7
        WHERE uuid = $8`,
		h.DbSchema)

	_, err = h.DbPool.Exec(
		ctx,
		updateQuery,
		payload.Email,
		payload.Username,
		payload.DisplayName,
		payload.FirstName,
		payload.LastName,
		payload.Locale,
		payload.Theme,
		userUuid,
	)
	if err != nil {
		debugf(err.Error())
		apiRequest.Error(http.StatusBadRequest, "Update user profile failed")
		return
	}

	apiRequest.SuccessNoData(http.StatusOK, "User profile updated successfully")
}

func (h *ApiHandler) AdminUpdateUserProfileSettings(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-update-user-profile-settings")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	var req struct {
		Locale *string `json:"locale"`
		Theme  *string `json:"theme"`
	}

	if err := gc.ShouldBindJSON(&req); err != nil {
		apiRequest.InvalidJSONInput()
		return
	}

	var setClauses []string
	var args []interface{}
	argPos := 1

	if req.Locale != nil {
		setClauses = append(setClauses, fmt.Sprintf("locale = $%d", argPos))
		args = append(args, *req.Locale)
		argPos++
	}

	if req.Theme != nil {
		setClauses = append(setClauses, fmt.Sprintf("theme = $%d", argPos))
		args = append(args, *req.Theme)
		argPos++
	}

	if len(setClauses) == 0 {
		apiRequest.Error(http.StatusBadRequest, "No fields provided")
		return
	}

	query := fmt.Sprintf(
		`UPDATE %s.user SET %s WHERE uuid = $%d`,
		h.DbSchema,
		strings.Join(setClauses, ", "),
		argPos,
	)

	args = append(args, userUuid)

	_, err := h.DbPool.Exec(ctx, query, args...)
	if err != nil {
		debugf(err.Error())
		apiRequest.Error(http.StatusBadRequest, "Update user settings failed")
		return
	}

	apiRequest.SuccessNoData(http.StatusOK, "User profile settings updated successfully")
}
