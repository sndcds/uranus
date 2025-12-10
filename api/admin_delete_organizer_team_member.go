package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// TODO: Review code

func (h *ApiHandler) AdminDeleteOrganizerTeamMember(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := h.DbPool

	organizerId, ok := ParamInt(gc, "organizerId")
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

	sql := fmt.Sprintf(`DELETE FROM %s.organizer_member_link WHERE organizer_id = $1 AND user_id = $2`, h.Config.DbSchema)
	_, err = tx.Exec(ctx, sql, organizerId, memberUserId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete team member: " + err.Error()})
		return
	}

	sql = fmt.Sprintf(`DELETE FROM %s.user_organizer_link WHERE organizer_id = $1 AND user_id = $2`, h.Config.DbSchema)
	_, err = tx.Exec(ctx, sql, organizerId, memberUserId)
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
