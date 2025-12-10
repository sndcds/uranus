package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// TODO: Review code

func (h *ApiHandler) AdminUpdateOrganizerMemberPermission(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := h.DbPool

	organizerId, ok := ParamInt(gc, "organizerId")
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "invalid organizerId"})
		return
	}

	memberId, ok := ParamInt(gc, "memberId")
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "invalid memberId"})
		return
	}

	// Parse incoming JSON: {"bit":24,"enabled":false}
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

	// Perform the bitwise update
	sql := fmt.Sprintf(`
		UPDATE %s.user_organizer_link
		SET permissions = CASE
			WHEN $1 THEN permissions | (1::bigint << $2)
			ELSE permissions & ~(1::bigint << $2)
		END
		WHERE user_id = $3 AND organizer_id = $4
		RETURNING permissions`,
		h.Config.DbSchema)

	fmt.Println(sql)

	var updatedPermissions int64
	err := pool.QueryRow(ctx, sql, input.Enabled, input.Bit, memberId, organizerId).Scan(&updatedPermissions)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			gc.JSON(http.StatusNotFound, gin.H{"error": "member not found"})
		} else {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	gc.JSON(http.StatusOK, gin.H{"permissions": updatedPermissions})
}
