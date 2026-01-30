package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
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

func (h *ApiHandler) AdminGetChoosableUserEventPlaces(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := h.userId(gc)

	query := app.UranusInstance.SqlAdminGetChoosableUserEventPlaces
	rows, err := h.DbPool.Query(ctx, query, userId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var places []model.EventPlace

	for rows.Next() {
		var place model.EventPlace
		err := rows.Scan(
			&place.VenueId,
			&place.VenueName,
			&place.SpaceId,
			&place.SpaceName,
			&place.City,
			&place.Country)
		if err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		places = append(places, place)
	}

	if err := rows.Err(); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(places) == 0 {
		gc.JSON(http.StatusOK, []model.EventPlace{}) // Returns empty array
		return
	}

	result := map[string]interface{}{
		"places":      places,
		"total_count": len(places),
	}

	gc.JSON(http.StatusOK, result)
}
