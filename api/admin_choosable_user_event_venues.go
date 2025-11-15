package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

// AdminChoosableUserEventVenues returns a list of event venues
// that can be selected (choosable) by an admin user. It responds with a JSON
// array of items.
//
// This endpoint is intended for administrative use only and may require
// authentication or specific permissions.
func (h *ApiHandler) AdminChoosableUserEventVenues(gc *gin.Context) {
	ctx := gc.Request.Context()
	db := h.DbPool
	userId := gc.GetInt("user-id")

	sql := app.Singleton.SqlAdminChoosableUserEventVenues
	rows, err := db.Query(ctx, sql, userId)
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
		if err := rows.Scan(
			&venue.Id,
			&venue.Name,
			&venue.City,
			&venue.CountryCode,
		); err != nil {
			fmt.Println(err.Error())
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
		// It's better to return an empty array instead of 204 so clients can safely parse it.
		gc.JSON(http.StatusOK, []Organizer{})
		return
	}

	gc.JSON(http.StatusOK, venues)
}
