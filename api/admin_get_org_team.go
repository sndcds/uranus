package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

func (h *ApiHandler) AdminGetOrgTeam(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-get-org-team")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	orgUuid := gc.Param("orgUuid")
	if orgUuid == "" {
		apiRequest.Error(http.StatusBadRequest, "orgUuid is required")
		return
	}

	members := []model.OrgMember{}
	invitedMembers := []model.InvitedOrgMember{}
	var canManagePermissions bool

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		permissions, err := h.GetUserOrgPermissionsTx(gc, tx, userUuid, orgUuid)
		if !permissions.Has(app.UserPermManageTeam) {
			return ApiErrForbidden("Insufficient permissions")
		}

		canManagePermissions = permissions.Has(app.UserPermManagePermissions)

		memberRows, err := tx.Query(ctx, app.UranusInstance.SqlAdminGetOrgMembers, orgUuid)
		if err != nil {
			return ApiErrInternal(err.Error())
		}
		defer memberRows.Close()

		for memberRows.Next() {
			var m model.OrgMember
			err := memberRows.Scan(
				&m.UserUuid,
				&m.Email,
				&m.Username,
				&m.DisplayName,
				&m.LastActiveAt,
				&m.JoinedAt,
			)
			if err != nil {
				return ApiErrInternal(err.Error())
			}

			m.AvatarUrl = h.getAvatarURL(m.UserUuid)
			members = append(members, m)
		}

		invitedMemberQuery := fmt.Sprintf(`
			SELECT
				oml.user_uuid,
				COALESCE(
					iu.display_name,
					NULLIF(CONCAT_WS(' ', iu.first_name, iu.last_name), ''),
				    iu.email
				) AS invited_by,
				oml.invited_at,
				u.email,
				u.display_name
			FROM %s.organization_member_link oml
			JOIN %s.user iu ON iu.uuid = oml.invited_by_user_uuid
			JOIN %s.user u ON u.uuid = oml.user_uuid
			WHERE oml.org_uuid = $1 AND has_joined = FALSE`,
			h.DbSchema, h.DbSchema, h.DbSchema)

		rows, err := tx.Query(ctx, invitedMemberQuery, orgUuid)
		if err != nil {
			return ApiErrInternal(err.Error())
		}
		defer rows.Close()

		for rows.Next() {
			var m model.InvitedOrgMember
			err = rows.Scan(&m.UserUuid, &m.InvitedBy, &m.InvitedAt, &m.Email, &m.DisplayName)
			if err != nil {
				return ApiErrInternal(err.Error())
			}

			m.AvatarUrl = h.getAvatarURL(m.UserUuid)
			invitedMembers = append(invitedMembers, m)
		}

		return nil
	})

	if txErr != nil {
		apiRequest.Error(txErr.Code, txErr.Error())
		return
	}

	result := gin.H{
		"members":                members,
		"invitations":            invitedMembers,
		"can_manage_permissions": canManagePermissions,
	}

	apiRequest.Success(http.StatusOK, result)
}
