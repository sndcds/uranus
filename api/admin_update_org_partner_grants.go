package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) AdminUpdateOrgPartnerGrants(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-get-org-partner-list")
	ctx := gc.Request.Context()

	orgUuid := gc.Param("orgUuid")
	if orgUuid == "" {
		apiRequest.Required("orgUuid is required")
		return
	}

	partnerUuid := gc.Param("partnerUuid")
	if partnerUuid == "" {
		apiRequest.Required("partnerUuid is required")
		return
	}

	type Payload struct {
		CanChooseVenue    bool `json:"can_choose_venue"`
		CanChoosePartner  bool `json:"can_choose_partner"`
		CanChoosePromoter bool `json:"can_choose_promoter"`
		CanSeeInsights    bool `json:"can_see_insights"`
	}
	var payload Payload

	if err := gc.ShouldBindJSON(&payload); err != nil {
		apiRequest.InvalidJSONInput()
		return
	}

	var permissions app.Permissions = 0
	if payload.CanChooseVenue {
		permissions |= app.OrgPermChooseVenue
	}
	if payload.CanChoosePartner {
		permissions |= app.OrgPermChoosePartner
	}
	if payload.CanChoosePromoter {
		permissions |= app.OrgPermChoosePromoter
	}
	if payload.CanSeeInsights {
		permissions |= app.OrgPermSeeInsights
	}

	query := fmt.Sprintf(`
		UPDATE %s.organization_access_grants
		SET permissions = $3
		WHERE src_org_uuid = $1 AND dst_org_uuid = $2`,
		h.DbSchema)
	_, err := h.DbPool.Exec(ctx, query, orgUuid, partnerUuid, permissions)
	if err != nil {
		apiRequest.InternalServerError()
		return
	}

	apiRequest.SuccessNoData(http.StatusOK, "")
}
