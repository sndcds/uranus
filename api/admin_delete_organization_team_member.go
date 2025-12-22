package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// TODO: Review code

func (h *ApiHandler) AdminDeleteOrganizationTeamMember(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := h.DbPool

	organizationId, ok := ParamInt(gc, "organizationId")
	if !ok {
		return
	}
	memberUserId, ok := ParamInt(gc, "memberId")
	if !ok {
		return
	}

	// Start a transaction
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction: " + err.Error()})
		return
	}
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	sql := fmt.Sprintf(`DELETE FROM %s.organization_member_link WHERE organization_id = $1 AND user_id = $2`, h.Config.DbSchema)
	_, err = tx.Exec(ctx, sql, organizationId, memberUserId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete team member: " + err.Error()})
		return
	}

	sql = fmt.Sprintf(`DELETE FROM %s.user_organization_link WHERE organization_id = $1 AND user_id = $2`, h.Config.DbSchema)
	_, err = tx.Exec(ctx, sql, organizationId, memberUserId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete team member: " + err.Error()})
		return
	}

	if err := tx.Commit(ctx); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit transaction: " + err.Error()})
		return
	}

	gc.JSON(http.StatusOK, gin.H{"message": "team member deleted"})
}
