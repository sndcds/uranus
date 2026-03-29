package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

// Permission note:
// - Caller must be authenticated
// - No explicit permission checks are performed in the handler
// - Authorization is enforced in the SQL query by filtering results using userId
//
// The query ensures that only venues and spaces accessible to the authenticated
// user are returned.
// Verified: 2026-01-11, Roald

func (h *ApiHandler) AdminGetChoosableUserEventVenues(gc *gin.Context) {
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)
	apiRequest := grains_api.NewRequest(gc, "choosable-user-event-venues")

	query := app.UranusInstance.SqlAdminGetChoosableUserEventVenues
	rows, err := h.DbPool.Query(ctx, query, userUuid)
	if err != nil {
		apiRequest.DatabaseError()
		return
	}
	defer rows.Close()

	var venueInfos []model.EventVenueInfo

	for rows.Next() {
		var venueInfo model.EventVenueInfo
		err := rows.Scan(
			&venueInfo.VenueId,
			&venueInfo.VenueName,
			&venueInfo.SpaceId,
			&venueInfo.SpaceName,
			&venueInfo.City,
			&venueInfo.Country)
		if err != nil {
			apiRequest.DatabaseError()
			return
		}
		venueInfos = append(venueInfos, venueInfo)
	}

	if err := rows.Err(); err != nil {
		apiRequest.DatabaseError()
		return
	}

	if len(venueInfos) == 0 {
		apiRequest.Success(http.StatusOK, []model.EventVenueInfo{}, "")
		return
	}

	result := map[string]interface{}{
		"venueInfos":  venueInfos,
		"total_count": len(venueInfos),
	}

	apiRequest.Success(http.StatusOK, result, "")
}
