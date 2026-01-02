package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

// AdminChoosableUserEventVenues returns a list of venues
// that can be selected (choosable) by user. It responds with a JSON
// array of items.
//
// This endpoint is intended for administrative use only and may require
// authentication or specific permissions.
func (h *ApiHandler) AdminChoosableUserEventVenues(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := gc.GetInt("user-id")

	query := app.UranusInstance.SqlAdminChoosableUserEventVenues
	rows, err := h.DbPool.Query(ctx, query, userId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type Venue struct {
		Id          int64   `json:"id"`
		Name        *string `json:"name"`
		City        *string `json:"city"`
		CountryCode *string `json:"country_code"`
	}

	var venues []Venue

	for rows.Next() {
		var venue Venue
		err := rows.Scan(&venue.Id, &venue.Name, &venue.City, &venue.CountryCode)
		if err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		venues = append(venues, venue)
	}

	if err := rows.Err(); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(venues) == 0 {
		gc.JSON(http.StatusOK, []Venue{}) // Returns empty array
		return
	}

	gc.JSON(http.StatusOK, venues)
}
