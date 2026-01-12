package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// TODO: Review code

func (h *ApiHandler) AdminDeleteOrganization(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := h.userId(gc)

	if !h.VerifyUserPassword(gc, userId) {
		return
	}

	organizationId, ok := ParamInt(gc, "organizationId")
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organizationId"})
		return
	}

	query := fmt.Sprintf(`DELETE FROM %s.organization WHERE id = $1`, h.Config.DbSchema)
	cmdTag, err := h.DbPool.Exec(ctx, query, organizationId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete organization", "details": err.Error()})
		return
	}

	if cmdTag.RowsAffected() == 0 {
		gc.JSON(http.StatusNotFound, gin.H{"error": "Organization not found"})
		return
	}

	gc.JSON(http.StatusOK, gin.H{"message": "Organization deleted successfully", "id": organizationId})
}
