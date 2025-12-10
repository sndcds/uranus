package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

// AdminChoosableUserEventOrganizers returns a list of event organizers
// that can be selected (choosable) by an admin user. It responds with a JSON
// array of items.
//
// This endpoint is intended for administrative use only and may require
// authentication or specific permissions.
func (h *ApiHandler) AdminChoosableUserEventOrganizers(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := gc.GetInt("user-id")

	// Parse organizer ID from path param
	organizerIdStr := gc.Param("organizerId")
	organizerId, err := strconv.Atoi(organizerIdStr)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := app.Singleton.SqlAdminChoosableUserEventOrganizers
	rows, err := h.DbPool.Query(ctx, query, userId, organizerId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type Organizer struct {
		Id          int64   `json:"id"`
		Name        *string `json:"name"`
		City        *string `json:"city"`
		CountryCode *string `json:"country_code"`
	}

	var organizers []Organizer

	for rows.Next() {
		var organizer Organizer
		err := rows.Scan(&organizer.Id, &organizer.Name, &organizer.City, &organizer.CountryCode)
		if err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		organizers = append(organizers, organizer)
	}

	if err := rows.Err(); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(organizers) == 0 {
		gc.JSON(http.StatusOK, []Organizer{}) // Returns empty array
		return
	}

	gc.JSON(http.StatusOK, organizers)
}
