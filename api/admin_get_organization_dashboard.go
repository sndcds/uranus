package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

type organizationListResponse struct {
	Organizations []model.OrganizationListItem `json:"organizations"`
}

// PermissionNote: User must be authenticated.
// The endpoint returns only organizations for which the authenticated user
// has a link in user_organization_link. The response includes the user’s
// permissions for each organization (edit, delete, manage team).
// PermissionChecks: Handled in SQL query; no additional checks needed here.
// Verified: 2026-01-12, Roald

func (h *ApiHandler) AdminGetOrganizationList(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-get-organization-list")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	rows, err := h.DbPool.Query(ctx, app.UranusInstance.SqlAdminGetOrganizationList, userUuid)
	if err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}
	defer rows.Close()

	var result organizationListResponse
	var userPermissions app.Permission
	for rows.Next() {
		var e model.OrganizationListItem
		if err := rows.Scan(
			&e.Uuid,
			&e.Name,
			&e.City,
			&e.Country,
			&e.TotalUpcomingEvents,
			&e.VenueCount,
			&e.SpaceCount,
			&userPermissions,
			&e.MainLogoUuid,
			&e.DarkThemeLogoUuid,
			&e.LightThemeLogoUuid,
		); err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		e.CanEditOrg = userPermissions.Has(app.PermEditOrganization)
		e.CanDeleteOrg = userPermissions.Has(app.PermDeleteOrganization)
		e.CanManageTeam = userPermissions.Has(app.PermManageTeam)
		result.Organizations = append(result.Organizations, e)
	}

	apiRequest.Success(http.StatusOK, result, "")
}
