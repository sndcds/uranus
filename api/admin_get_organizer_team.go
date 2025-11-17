package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) AdminGetOrganizerTeam(gc *gin.Context) {
	pool := h.DbPool
	ctx := gc.Request.Context()

	// Get organizer ID
	organizerIdStr := gc.Param("organizerId")
	organizerId, err := strconv.Atoi(organizerIdStr)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "invalid organizer id"})
		return
	}

	langStr := gc.DefaultQuery("lang", "en")

	// Start a transaction
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback(ctx) // safe rollback if not committed

	// --- Fetch members ---
	memberSQL := fmt.Sprintf(`
        SELECT
            u.id AS user_id,
            u.email_address AS email,
            u.user_name,
            COALESCE(u.display_name, u.first_name || ' ' || u.last_name) AS display_name,
            u.modified_at AS last_active_at,
            uml.created_at AS joined_at
        FROM %s.organizer_member_link uml
        JOIN %s."user" u ON u.id = uml.user_id
        WHERE uml.organizer_id = $1 AND u.user_name IS NOT NULL AND uml.has_joined = true
        ORDER BY display_name`,
		h.Config.DbSchema, h.Config.DbSchema) // TODO: Prepare

	memberRows, err := tx.Query(ctx, memberSQL, organizerId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer memberRows.Close()

	type OrganizerMember struct {
		UserId       int        `json:"user_id"`
		Email        string     `json:"email"`
		UserName     *string    `json:"user_name"`
		DisplayName  *string    `json:"display_name"`
		AvatarUrl    *string    `json:"avatar_url"`
		LastActiveAt *time.Time `json:"last_active_at"`
		JoinedAt     time.Time  `json:"joined_at"`
	}

	members := []OrganizerMember{}

	for memberRows.Next() {
		var m OrganizerMember
		err := memberRows.Scan(
			&m.UserId,
			&m.Email,
			&m.UserName,
			&m.DisplayName,
			&m.LastActiveAt,
			&m.JoinedAt,
		)
		if err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
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

	type InvitedMember struct {
		UserID    int       `json:"user_id"`
		InvitedBy string    `json:"invited_by"`
		InvitedAt time.Time `json:"invited_at"`
		Email     string    `json:"email"`
		RoleID    int       `json:"role_id"`
		RoleName  string    `json:"role_name"`
	}
	invitedMemberSql := fmt.Sprintf(`
		SELECT
			oml.user_id,
			COALESCE(iu.display_name, iu.first_name || ' ' || iu.last_name) AS invited_by,
			oml.invited_at,
			u.email_address AS email,
			oml.member_role_id AS role_id,
			tmr.name AS role_name
		FROM %s.organizer_member_link oml
		JOIN %s.user iu ON iu.id = oml.invited_by_user_id
		JOIN %s.user u ON u.id = oml.user_id
		JOIN %s.team_member_role tmr ON tmr.type_id = oml.member_role_id AND tmr.iso_639_1 = $2
		WHERE oml.organizer_id = $1 AND has_joined = false`,
		h.Config.DbSchema, h.Config.DbSchema, h.Config.DbSchema, h.Config.DbSchema)
	rows, err := tx.Query(ctx, invitedMemberSql, organizerId, langStr)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch invited members" + err.Error()})
		return
	}
	defer rows.Close()

	var invitedMembers []InvitedMember
	for rows.Next() {
		var m InvitedMember
		if err := rows.Scan(&m.UserID, &m.InvitedBy, &m.InvitedAt, &m.Email, &m.RoleID, &m.RoleName); err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to scan invited member"})
			return
		}
		invitedMembers = append(invitedMembers, m)
	}

	// --- Fetch roles ---
	rolesSQL := fmt.Sprintf(`
        SELECT type_id AS id, name, description
        FROM %s.team_member_role
        WHERE iso_639_1 = $1
        ORDER BY id;
    `, h.Config.DbSchema)

	fmt.Println("rolesSQL: ", rolesSQL)
	roleRows, err := tx.Query(ctx, rolesSQL, langStr)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer roleRows.Close()

	type Role struct {
		Id          int    `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	roles := []Role{}
	for roleRows.Next() {
		var role Role
		if err := roleRows.Scan(&role.Id, &role.Name, &role.Description); err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		roles = append(roles, role)
		fmt.Println(role)
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return combined JSON
	result := gin.H{
		"members":     members,
		"invitations": invitedMembers,
		"roles":       roles,
	}

	b, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(b)) // optional: print to console

	gc.JSON(http.StatusOK, result)
}
