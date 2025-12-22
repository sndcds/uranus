package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

// TODO: Review code

type organizationDashboardEntry struct {
	OrganizationId          int64   `json:"organization_id"`
	OrganizationName        string  `json:"organization_name"`
	OrganizationCity        *string `json:"organization_city"`
	OrganizationCountryCode *string `json:"organization_country_code"`
	TotalUpcomingEvents     int64   `json:"total_upcoming_events"`
	VenueCount              int64   `json:"venue_count"`
	SpaceCount              int64   `json:"space_count"`
	CanEditOrganization     bool    `json:"can_edit_organization"`
	CanDeleteOrganization   bool    `json:"can_delete_organization"`
	CanManageTeam           bool    `json:"can_manage_team"`
}

type organizationDashboardResponse struct {
	Organizations []organizationDashboardEntry `json:"organizations"`
}

func (h *ApiHandler) AdminGetOrganizationDashboard(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := gc.GetInt("user-id")

	rows, err := h.DbPool.Query(ctx, app.Singleton.SqlAdminGetOrganizationDashboard, userId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var result organizationDashboardResponse
	var userPermissions app.Permission
	for rows.Next() {
		var e organizationDashboardEntry
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
