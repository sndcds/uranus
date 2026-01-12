package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

// TODO: Review code

type organizationDashboardResponse struct {
	Organizations []model.OrganizationDashboardEntry `json:"organizations"`
}

// PermissionNote: User must be authenticated.
// The endpoint returns only organizations for which the authenticated user
// has a link in user_organization_link. The response includes the user’s
// permissions for each organization (edit, delete, manage team).
// PermissionChecks: Handled in SQL query; no additional checks needed here.
// Verified: 2026-01-12, Roald

func (h *ApiHandler) AdminGetOrganizationDashboard(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := h.userId(gc)

	rows, err := h.DbPool.Query(ctx, app.UranusInstance.SqlAdminGetOrganizationDashboard, userId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var result organizationDashboardResponse
	var userPermissions app.Permission
	for rows.Next() {
		var e model.OrganizationDashboardEntry
		if err := rows.Scan(
			&e.OrganizationId,
			&e.OrganizationName,
			&e.OrganizationCity,
			&e.OrganizationCountryCode,
			&e.TotalUpcomingEvents,
			&e.VenueCount,
			&e.SpaceCount,
			&userPermissions,
		); err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		e.CanEditOrganization = userPermissions.Has(app.PermEditOrganization)
		e.CanDeleteOrganization = userPermissions.Has(app.PermDeleteOrganization)
		e.CanManageTeam = userPermissions.Has(app.PermManageTeam)
		result.Organizations = append(result.Organizations, e)
	}

	gc.JSON(http.StatusOK, result)
}
