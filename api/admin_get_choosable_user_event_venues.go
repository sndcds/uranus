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
	apiRequest := grains_api.NewRequest(gc, "choosable-user-event-venues")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	query := app.UranusInstance.SqlAdminChoosableUserEventVenues
	debugf(query)
	debugf(userUuid)
	rows, err := h.DbPool.Query(ctx, query, userUuid)
	if err != nil {
		debugf(err.Error())
		apiRequest.DatabaseError()
		return
	}
	defer rows.Close()

	var venueInfos []model.VenueInfo

	for rows.Next() {
		var venueInfo model.VenueInfo
		err := rows.Scan(
			&venueInfo.VenueUuid,
			&venueInfo.VenueName,
			&venueInfo.SpaceUuid,
			&venueInfo.SpaceName,
			&venueInfo.City,
			&venueInfo.Country)
		if err != nil {
			debugf(err.Error())
			apiRequest.DatabaseError()
			return
		}
		venueInfos = append(venueInfos, venueInfo)
	}

	err = rows.Err()
	if err != nil {
		debugf(err.Error())
		apiRequest.DatabaseError()
		return
	}

	if len(venueInfos) == 0 {
		apiRequest.Success(http.StatusOK, []model.VenueInfo{}, "")
		return
	}

	result := map[string]interface{}{
		"venue_infos": venueInfos,
		"total_count": len(venueInfos),
	}

	apiRequest.Success(http.StatusOK, result, "")
}
