package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

// Permission note:
// - Caller must be authenticated
// - No explicit permission checks are performed in the handler
// - Authorization is enforced in the SQL query by filtering results using userId
//
// The query ensures that only venues and spaces accessible to the authenticated
// user are returned.
// Verified: 2026-01-11, Roald

func (h *ApiHandler) AdminChoosableUserVenuesSpaces(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := h.userId(gc)

	query := app.UranusInstance.SqlAdminChoosableUserVenuesSpaces
	rows, err := h.DbPool.Query(ctx, query, userId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type Place struct {
		VenueId     int64   `json:"venue_id"`
		VenueName   *string `json:"venue_name"`
		SpaceId     *int64  `json:"space_id"`
		SpaceName   *string `json:"space_name"`
		City        *string `json:"city"`
		CountryCode *string `json:"country_code"`
	}

	var places []Place

	for rows.Next() {
		var place Place
		err := rows.Scan(
			&place.VenueId,
			&place.VenueName,
			&place.SpaceId,
			&place.SpaceName,
			&place.City,
			&place.CountryCode)
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
		gc.JSON(http.StatusOK, []Place{}) // Returns empty array
		return
	}

	gc.JSON(http.StatusOK, places)
}
