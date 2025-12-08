package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) AdminChoosableUserVenuesSpaces(gc *gin.Context) {
	ctx := gc.Request.Context()
	db := h.DbPool
	userId := gc.GetInt("user-id")

	sql := app.Singleton.SqlAdminChoosableUserVenuesSpaces
	rows, err := db.Query(ctx, sql, userId)
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
		if err := rows.Scan(
			&place.VenueId,
			&place.VenueName,
			&place.SpaceId,
			&place.SpaceName,
			&place.City,
			&place.CountryCode,
		); err != nil {
			fmt.Println(err.Error())
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
		// It's better to return an empty array instead of 204 so clients can safely parse it.
		gc.JSON(http.StatusOK, []Place{})
		return
	}

	gc.JSON(http.StatusOK, places)
}
