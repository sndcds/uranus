package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

func (h *ApiHandler) AdminGetOrgPartnerGrants(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-get-org-partner-list")
	ctx := gc.Request.Context()

	orgUuid := gc.Param("orgUuid")
	if orgUuid == "" {
		apiRequest.Required("orgUuid is required")
		return
	}

	rows, err := h.DbPool.Query(ctx, app.UranusInstance.SqlAdminGetOrgPartnerList, orgUuid)
	if err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}
	defer rows.Close()

	type Response struct {
		Partners []model.OrgPartnerListItem `json:"partner_grants"`
	}

	var result Response
	var permissions app.Permissions

	for rows.Next() {
		var p model.OrgPartnerListItem
		if err := rows.Scan(
			&p.OrgUuid,
			&p.OrgName,
			&permissions,
			&p.Direction,
		); err != nil {
			debugf(err.Error())
			apiRequest.InternalServerError()
			return
		}

		p.CanChooseVenue = permissions.Has(app.OrgPermChooseVenue)
		p.CanChoosePartner = permissions.Has(app.OrgPermChoosePartner)
		p.CanChoosePromoter = permissions.Has(app.OrgPermChoosePromoter)
		p.CanSeeInsights = permissions.Has(app.OrgPermSeeInsights)

		// TODO: Get partner logos

		result.Partners = append(result.Partners, p)
	}

	apiRequest.Success(http.StatusOK, result, "")
}
