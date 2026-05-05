package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

func (h *ApiHandler) AdminGetOrgPartnerRequest(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-get-org-partner-requests")
	ctx := gc.Request.Context()

	orgUuid := gc.Param("orgUuid")
	if orgUuid == "" {
		apiRequest.Required("orgUuid is required")
		return
	}

	rows, err := h.DbPool.Query(ctx, app.UranusInstance.SqlAdminGetOrgPartnerRequests, orgUuid)
	if err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}
	defer rows.Close()

	type Response struct {
		Partners []model.OrgPartnerRequestItem `json:"partner_requests"`
	}

	var result Response

	for rows.Next() {
		var p model.OrgPartnerRequestItem

		if err := rows.Scan(
			&p.OrgUuid,
			&p.OrgName,
			&p.CreatedAt,
			&p.Message,
			&p.Direction,
			&p.Status,
		); err != nil {
			debugf(err.Error())
			apiRequest.InternalServerError()
			return
		}

		result.Partners = append(result.Partners, p)
	}

	apiRequest.Success(http.StatusOK, result, "")
}
