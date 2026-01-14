package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

func (h *ApiHandler) AdminGetOrganizationTeam(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := h.userId(gc)

	organizationId, ok := ParamInt(gc, "organizationId")
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization id"})
		return
	}

	members := []model.OrganizationMember{}
	invitedMembers := []model.InvitedOrganizationMember{}

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {

		txErr := h.CheckOrganizationPermission(gc, tx, userId, organizationId, app.PermManageTeam)
		if txErr != nil {
			return txErr
		}

		// Fetch members
		memberRows, err := tx.Query(ctx, app.UranusInstance.SqlAdminGetOrganizationMembers, organizationId)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("Transaction failed: %s", err.Error()),
			}
		}
		defer memberRows.Close()

		for memberRows.Next() {
			var m model.OrganizationMember
			err := memberRows.Scan(
				&m.MemberId,
				&m.UserId,
				&m.Email,
				&m.UserName,
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

			// Optional: Add avatar URL if file exists
			imageDir := app.UranusInstance.Config.ProfileImageDir
			imagePath := filepath.Join(imageDir, fmt.Sprintf("profile_img_%d_64.webp", m.UserId))
			if _, err := os.Stat(imagePath); err == nil {
				avatarUrl := fmt.Sprintf(`%s/api/user/%d/avatar/64`, h.Config.BaseApiUrl, m.UserId)
				m.AvatarUrl = &avatarUrl
			}

			members = append(members, m)
		}

		invitedMemberQuery := fmt.Sprintf(`
		SELECT
			oml.user_id,
			COALESCE(iu.display_name, iu.first_name || ' ' || iu.last_name) AS invited_by,
			oml.invited_at,
			u.email_address AS email
		FROM %s.organization_member_link oml
		JOIN %s.user iu ON iu.id = oml.invited_by_user_id
		JOIN %s.user u ON u.id = oml.user_id
		WHERE oml.organization_id = $1 AND has_joined = false`,
			h.Config.DbSchema, h.Config.DbSchema, h.Config.DbSchema)
		fmt.Println(invitedMemberQuery)
		rows, err := tx.Query(ctx, invitedMemberQuery, organizationId)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("Failed to fetch invited members: %s", err.Error()),
			}
		}
		defer rows.Close()

		for rows.Next() {
			var m model.InvitedOrganizationMember
			err = rows.Scan(&m.UserID, &m.InvitedBy, &m.InvitedAt, &m.Email)
			if err != nil {
				return &ApiTxError{
					Code: http.StatusInternalServerError,
					Err:  fmt.Errorf("Failed to scan invited members: %s", err.Error()),
				}
			}
			invitedMembers = append(invitedMembers, m)
		}

		return nil
	})
	if txErr != nil {
		gc.JSON(txErr.Code, gin.H{"error": txErr.Error()})
		return
	}

	result := gin.H{
		"members":     members,
		"invitations": invitedMembers,
	}

	gc.JSON(http.StatusOK, result)
}
