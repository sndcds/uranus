package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// TODO: Review code

func (h *ApiHandler) AdminDeleteSpace(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := h.userId(gc)

	if !h.VerifyUserPassword(gc, userId) {
		return
	}

	spaceId, ok := ParamInt(gc, "spaceId")
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "Invalid space Id"})
		return
	}
	
	query := fmt.Sprintf(`DELETE FROM %s.space WHERE id = $1`, h.Config.DbSchema)
	cmdTag, err := h.DbPool.Exec(ctx, query, spaceId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete space", "details": err.Error()})
		return
	}

	if cmdTag.RowsAffected() == 0 {
		gc.JSON(http.StatusNotFound, gin.H{"error": "Space not found"})
		return
	}

	gc.JSON(http.StatusOK, gin.H{"message": "Space deleted successfully", "id": spaceId})
}
