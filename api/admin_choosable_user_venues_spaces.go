package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) AdminChoosableUserVenuesSpaces(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := gc.GetInt("user-id")

	query := app.Singleton.SqlAdminChoosableUserVenuesSpaces
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
