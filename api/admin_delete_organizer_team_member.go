package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func (h *ApiHandler) AdminDeleteOrganizerTeamMember(gc *gin.Context) {
	pool := h.DbPool
	ctx := gc.Request.Context()

	memberUserIdStr := gc.Param("memberUserId")
	memberUserId, err := strconv.Atoi(memberUserIdStr)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "invalid member user ID"})
		return
	}

	sql := fmt.Sprintf(`DELETE FROM %s.organizer_member_link WHERE user_id = $1`, h.Config.DbSchema)

	rows, err := pool.Query(ctx, sql, memberUserId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	gc.JSON(http.StatusOK, gin.H{"message": "team member deleted"})
}
