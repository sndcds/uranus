package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

// PermissionNote: User must be authenticated.
// The endpoint returns only organizations for which the authenticated user
// has a link in user_organization_link. The response includes the user’s
// permissions for each organization (edit, delete, manage team).
// PermissionChecks: Handled in SQL query; no additional checks needed here.
// Verified: 2026-01-12, Roald

func (h *ApiHandler) AdminGetOrganizationList(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-get-org-list")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	rows, err := h.DbPool.Query(ctx, app.UranusInstance.SqlAdminGetOrgList, userUuid)
	if err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}
	defer rows.Close()

	var logoUuid *string
	var lightThemeLogoUuid *string
	var darkThemeLogoUuid *string

	type Response struct {
		Organizations []model.OrgListItem `json:"organizations"`
	}

	var result Response
	var userPermissions app.Permissions
	for rows.Next() {
		var e model.OrgListItem
		if err := rows.Scan(
			&e.Uuid,
			&e.Name,
			&e.City,
			&e.Country,
			&e.TotalUpcomingEvents,
			&e.VenueCount,
			&e.SpaceCount,
			&userPermissions,
			&logoUuid,
			&lightThemeLogoUuid,
			&darkThemeLogoUuid,
		); err != nil {
			apiRequest.InternalServerError()
			return
		}

		if logoUuid != nil {
			url := ImageUrl(*logoUuid)
			e.LogoUrl = &url
		}
		if lightThemeLogoUuid != nil {
			url := ImageUrl(*lightThemeLogoUuid)
			e.LightThemeLogoUrl = &url
		}
		if darkThemeLogoUuid != nil {
			url := ImageUrl(*darkThemeLogoUuid)
			e.DarkThemeLogoUrl = &url
		}

		e.CanEditOrg = userPermissions.Has(app.UserPermEditOrg)
		e.CanDeleteOrg = userPermissions.Has(app.UserPermDeleteOrg)
		e.CanManageTeam = userPermissions.Has(app.UserPermManageTeam)

		result.Organizations = append(result.Organizations, e)
	}

	apiRequest.Success(http.StatusOK, result, "")
}
