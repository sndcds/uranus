package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

// TODO: Review code

type organizerDashboardEntry struct {
	OrganizerId          int64   `json:"organizer_id"`
	OrganizerName        string  `json:"organizer_name"`
	OrganizerCity        *string `json:"organizer_city"`
	OrganizerCountryCode *string `json:"organizer_country_code"`
	TotalUpcomingEvents  int64   `json:"total_upcoming_events"`
	VenueCount           int64   `json:"venue_count"`
	SpaceCount           int64   `json:"space_count"`
	CanEditOrganizer     bool    `json:"can_edit_organizer"`
	CanDeleteOrganizer   bool    `json:"can_delete_organizer"`
	CanManageTeam        bool    `json:"can_manage_team"`
}

type organizerDashboardResponse struct {
	Organizers []organizerDashboardEntry `json:"organizers"`
}

func (h *ApiHandler) AdminGetOrganizerDashboard(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := gc.GetInt("user-id")

	rows, err := h.DbPool.Query(ctx, app.Singleton.SqlAdminGetOrganizerDashboard, userId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var result organizerDashboardResponse
	var userPermissions app.Permission
	for rows.Next() {
		var e organizerDashboardEntry
		if err := rows.Scan(
			&e.OrganizerId,
			&e.OrganizerName,
			&e.OrganizerCity,
			&e.OrganizerCountryCode,
			&e.TotalUpcomingEvents,
			&e.VenueCount,
			&e.SpaceCount,
			&userPermissions,
		); err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		e.CanEditOrganizer = userPermissions.Has(app.PermEditOrganizer)
		e.CanDeleteOrganizer = userPermissions.Has(app.PermDeleteOrganizer)
		e.CanManageTeam = userPermissions.Has(app.PermManageTeam)
		result.Organizers = append(result.Organizers, e)
	}

	gc.JSON(http.StatusOK, result)
}
