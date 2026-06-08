package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

func (h *ApiHandler) AdminGetOrgPartnerGrants(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-get-org-partner-list")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	orgUuid := gc.Param("orgUuid")
	if orgUuid == "" {
		apiRequest.Required("orgUuid is required")
		return
	}

	type Response struct {
		CanRequestPartner        bool                       `json:"request_partner"`
		CanAnswerPartnerRequests bool                       `json:"answer_partner_requests"`
		CanEditPartnerRights     bool                       `json:"edit_partner_rights"`
		CanDeletePartnership     bool                       `json:"delete_partnership"`
		Partners                 []model.OrgPartnerListItem `json:"partner_grants"`
	}
	var result Response

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		orgPermissions, err := h.GetUserOrgPermissionsTx(gc, tx, userUuid, orgUuid)
		if err != nil {
			return TxInternalError(err)
		}

		result.CanRequestPartner = orgPermissions.Has(app.UserPermRequestPartner)
		result.CanAnswerPartnerRequests = orgPermissions.Has(app.UserPermAnswerPartnerRequest)
		result.CanEditPartnerRights = orgPermissions.Has(app.UserPermEditPartnerRights)
		result.CanDeletePartnership = orgPermissions.Has(app.UserPermDeletePartnership)

		rows, err := tx.Query(ctx, app.UranusInstance.SqlAdminGetOrgPartnerList, orgUuid)
		if err != nil {
			return TxInternalError(err)
		}
		defer rows.Close()

		var partnerPermissions app.Permissions

		for rows.Next() {
			var p model.OrgPartnerListItem
			if err := rows.Scan(
				&p.OrgUuid,
				&p.OrgName,
				&partnerPermissions,
				&p.Direction,
			); err != nil {
				return TxInternalError(err)
			}

			p.CanChooseVenue = partnerPermissions.Has(app.OrgPermChooseVenue)
			p.CanChoosePartner = partnerPermissions.Has(app.OrgPermChoosePartner)
			p.CanChoosePromoter = partnerPermissions.Has(app.OrgPermChoosePromoter)
			p.CanSeeInsights = partnerPermissions.Has(app.OrgPermSeeInsights)

			// TODO: Get partner logos

			result.Partners = append(result.Partners, p)
		}

		return nil
	})

	if txErr != nil {
		debugf(txErr.Error())
		apiRequest.Error(txErr.Code, txErr.Error())
		return
	}

	apiRequest.Success(http.StatusOK, result)
}
