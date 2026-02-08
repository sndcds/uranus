package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grainsapi"
)

func (h *ApiHandler) AdminDeleteOrganization(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := h.userId(gc)
	apiRequest := grainsapi.NewRequest(gc, "admin-delete-organization")

	err := h.VerifyUserPassword(gc, userId)
	if err != nil {
		apiRequest.Error(http.StatusUnauthorized, err.Error())
		return
	}

	orgId, ok := ParamInt(gc, "organizationId")
	if !ok {
		apiRequest.Error(http.StatusBadRequest, "invalid organizationId")
		return
	}
	apiRequest.SetMeta("organization_id", orgId)

	query := fmt.Sprintf(`DELETE FROM %s.organization WHERE id = $1`, h.Config.DbSchema)
	cmdTag, err := h.DbPool.Exec(ctx, query, orgId)
	if err != nil {
		apiRequest.Error(http.StatusInternalServerError, "failed to delete organization")
		return
	}

	if cmdTag.RowsAffected() == 0 {
		apiRequest.Error(http.StatusNotFound, "organization not found")
		return
	}

	apiRequest.SuccessNoData(http.StatusOK, "organization deleted successfully")
}
