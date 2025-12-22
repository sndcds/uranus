package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) AdminUpdateOrganizationMemberPermission(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := gc.GetInt("user-id")

	organizationId, ok := ParamInt(gc, "organizationId")
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "organization Id is required"})
		return
	}

	memberId, ok := ParamInt(gc, "memberId")
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "memberID is required"})
		return
	}

	var input struct {
		Bit     int  `json:"bit"`
		Enabled bool `json:"enabled"`
	}
	if err := gc.ShouldBindJSON(&input); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}
	if input.Bit < 0 || input.Bit > 63 {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "bit must be between 0 and 63"})
		return
	}

	// Begin transaction
	tx, err := h.DbPool.Begin(ctx)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Check if user can manage member permissions as the organization
	organizationPermissions, err := h.GetUserOrganizationPermissions(gc, tx, userId, organizationId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !organizationPermissions.HasAll(app.PermManagePermissions | app.PermManageTeam) {
		gc.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
		return
	}

	// Perform the bitwise update
	query := fmt.Sprintf(`
		UPDATE %s.user_organization_link
		SET permissions = CASE
			WHEN $1 THEN permissions | (1::bigint << $2)
			ELSE permissions & ~(1::bigint << $2)
		END
		WHERE user_id = $3 AND organization_id = $4
		RETURNING permissions`,
		h.Config.DbSchema)

	var updatedPermissions int64
	err = tx.QueryRow(ctx, query, input.Enabled, input.Bit, memberId, organizationId).
		Scan(&updatedPermissions)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			gc.JSON(http.StatusNotFound, gin.H{"error": "member not found"})
			return
		}
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err = tx.Commit(ctx); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to commit transaction: %v", err)})
		return
	}

	gc.JSON(http.StatusOK, gin.H{"permissions": updatedPermissions})
}
