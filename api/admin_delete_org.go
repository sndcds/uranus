package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
)

func (h *ApiHandler) AdminDeleteOrganization(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-delete-org")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	err := h.VerifyUserPassword(gc, userUuid)
	if err != nil {
		apiRequest.Error(http.StatusUnauthorized, err.Error())
		return
	}

	orgUuid := gc.Param("orgUuid")
	if orgUuid == "" {
		apiRequest.Required("orgUuid is required")
		return
	}
	apiRequest.SetMeta("org_uuid", orgUuid)

	query := fmt.Sprintf(`DELETE FROM %s.organization WHERE uuid = $1::uuid`, h.DbSchema)
	cmdTag, err := h.DbPool.Exec(ctx, query, orgUuid)
	if err != nil {
		debugf(err.Error())
		apiRequest.Error(http.StatusInternalServerError, "failed to delete organization")
		return
	}

	if cmdTag.RowsAffected() == 0 {
		apiRequest.Error(http.StatusNotFound, "organization not found")
		return
	}

	apiRequest.SuccessNoData(http.StatusOK, "organization deleted successfully")
}
