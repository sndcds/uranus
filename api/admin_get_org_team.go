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
		apiRequest.Error(http.StatusBadRequest, "invalid orgUuid")
		return
	}

	members := []model.OrgMember{}
	invitedMembers := []model.InvitedOrgMember{}

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		txErr := h.CheckOrganizationPermissionTx(gc, tx, userUuid, orgUuid, app.UserPermManageTeam)
		if txErr != nil {
			return txErr
		}

		// Fetch members
		memberRows, err := tx.Query(ctx, app.UranusInstance.SqlAdminGetOrgMembers, orgUuid)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("Transaction failed: %s", err.Error()),
			}
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
				return &ApiTxError{
					Code: http.StatusInternalServerError,
					Err:  fmt.Errorf("Transaction failed: %s", err.Error()),
				}
			}

			m.AvatarUrl = h.getAvatarURL(m.UserUuid)
			members = append(members, m)
		}

		invitedMemberQuery := fmt.Sprintf(`
			SELECT
				oml.user_uuid,
				COALESCE(iu.display_name, iu.first_name || ' ' || iu.last_name) AS invited_by,
				oml.invited_at,
				u.email,
				u.display_name
			FROM %s.organization_member_link oml
			JOIN %s.user iu ON iu.uuid = oml.invited_by_user_uuid
			JOIN %s.user u ON u.uuid = oml.user_uuid
			WHERE oml.org_uuid = $1 AND has_joined = false`,
			h.DbSchema, h.DbSchema, h.DbSchema)

		rows, err := tx.Query(ctx, invitedMemberQuery, orgUuid)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("failed to fetch invited members: %s", err.Error()),
			}
		}
		defer rows.Close()

		for rows.Next() {
			var m model.InvitedOrgMember
			err = rows.Scan(&m.UserUuid, &m.InvitedBy, &m.InvitedAt, &m.Email, &m.DisplayName)
			if err != nil {
				return &ApiTxError{
					Code: http.StatusInternalServerError,
					Err:  fmt.Errorf("failed to scan invited members: %s", err.Error()),
				}
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
		"members":     members,
		"invitations": invitedMembers,
	}

	apiRequest.Success(http.StatusOK, result, "")
}
