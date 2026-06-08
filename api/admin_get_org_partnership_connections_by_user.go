package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) AdminOrgPartnershipConnectionsByUser(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-get-org-partnership-connections-by-user")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	query := app.UranusInstance.SqlAdminGetOrgPartnershipConnectionsByUser

	rows, err := h.DbPool.Query(ctx, query, userUuid)
	if err != nil {
		apiRequest.InternalServerError()
		return
	}
	defer rows.Close()

	result := struct {
		OrgAccessGrants []struct {
			SrcOrgUuid  string `json:"src_org_uuid"`
			DstOrgUuid  string `json:"dst_org_uuid"`
			SrcOrgName  string `json:"src_org_name"`
			DstOrgName  string `json:"dst_org_name"`
			Permissions int64  `json:"permissions"`
			SrcAccess   bool   `json:"src_access"`
			DstAccess   bool   `json:"dst_access"`
		} `json:"org_access_grants"`
	}{
		OrgAccessGrants: make([]struct {
			SrcOrgUuid  string `json:"src_org_uuid"`
			DstOrgUuid  string `json:"dst_org_uuid"`
			SrcOrgName  string `json:"src_org_name"`
			DstOrgName  string `json:"dst_org_name"`
			Permissions int64  `json:"permissions"`
			SrcAccess   bool   `json:"src_access"`
			DstAccess   bool   `json:"dst_access"`
		}, 0),
	}

	for rows.Next() {
		var g struct {
			SrcOrgUuid  string
			DstOrgUuid  string
			SrcOrgName  string
			DstOrgName  string
			Permissions int64
			SrcAccess   bool
			DstAccess   bool
		}

		if err := rows.Scan(
			&g.SrcOrgUuid,
			&g.DstOrgUuid,
			&g.SrcOrgName,
			&g.DstOrgName,
			&g.Permissions,
			&g.SrcAccess,
			&g.DstAccess,
		); err != nil {
			apiRequest.InternalServerError()
			return
		}

		result.OrgAccessGrants = append(result.OrgAccessGrants, struct {
			SrcOrgUuid  string `json:"src_org_uuid"`
			DstOrgUuid  string `json:"dst_org_uuid"`
			SrcOrgName  string `json:"src_org_name"`
			DstOrgName  string `json:"dst_org_name"`
			Permissions int64  `json:"permissions"`
			SrcAccess   bool   `json:"src_access"`
			DstAccess   bool   `json:"dst_access"`
		}{
			SrcOrgUuid:  g.SrcOrgUuid,
			DstOrgUuid:  g.DstOrgUuid,
			SrcOrgName:  g.SrcOrgName,
			DstOrgName:  g.DstOrgName,
			Permissions: g.Permissions,
			SrcAccess:   g.SrcAccess,
			DstAccess:   g.DstAccess,
		})
	}

	apiRequest.Success(http.StatusOK, result)
}
