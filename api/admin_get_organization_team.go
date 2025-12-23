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

	organizationId, ok := ParamInt(gc, "organizationId")
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization id"})
		return
	}

	langStr := gc.DefaultQuery("lang", "en")

	members := []model.OrganizationMember{}
	var invitedMembers []model.InvitedOrganizationMember
	memberRoles := []model.OrganizationMemberRole{}

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {

		// --- Fetch members ---
		memberRows, err := tx.Query(ctx, app.Singleton.SqlAdminGetOrganizationMembers, organizationId)
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

			// Optional: add avatar URL if file exists
			imageDir := app.Singleton.Config.ProfileImageDir
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
			u.email_address AS email,
			oml.member_role_id AS role_id,
			tmr.name AS role_name
		FROM %s.organization_member_link oml
		JOIN %s.user iu ON iu.id = oml.invited_by_user_id
		JOIN %s.user u ON u.id = oml.user_id
		JOIN %s.team_member_role tmr ON tmr.type_id = oml.member_role_id AND tmr.iso_639_1 = $2
		WHERE oml.organization_id = $1 AND has_joined = false`,
			h.Config.DbSchema, h.Config.DbSchema, h.Config.DbSchema, h.Config.DbSchema)
		rows, err := tx.Query(ctx, invitedMemberQuery, organizationId, langStr)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("Failed to fetch invited members: %s", err.Error()),
			}
		}
		defer rows.Close()

		for rows.Next() {
			var m model.InvitedOrganizationMember
			err = rows.Scan(&m.UserID, &m.InvitedBy, &m.InvitedAt, &m.Email, &m.RoleID, &m.RoleName)
			if err != nil {
				return &ApiTxError{
					Code: http.StatusInternalServerError,
					Err:  fmt.Errorf("Failed to scan invited members: %s", err.Error()),
				}
			}
			invitedMembers = append(invitedMembers, m)
		}

		// --- Fetch roles ---
		rolesQuery := fmt.Sprintf(`
        SELECT type_id AS id, name, description
        FROM %s.team_member_role
        WHERE iso_639_1 = $1
        ORDER BY id;
    `, h.Config.DbSchema)

		roleRows, err := tx.Query(ctx, rolesQuery, langStr)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("Transaction failed: %s", err.Error()),
			}
		}
		defer roleRows.Close()

		for roleRows.Next() {
			var role model.OrganizationMemberRole
			if err := roleRows.Scan(&role.Id, &role.Name, &role.Description); err != nil {
				return &ApiTxError{
					Code: http.StatusInternalServerError,
					Err:  fmt.Errorf("Transaction failed: %s", err.Error()),
				}
			}
			memberRoles = append(memberRoles, role)
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
		"roles":       memberRoles,
	}

	gc.JSON(http.StatusOK, result)
}
