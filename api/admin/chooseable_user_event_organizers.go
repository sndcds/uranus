package api_admin

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

func ChoosableUserEventOrganizersHandler(gc *gin.Context) {
	db := app.Singleton.MainDbPool
	ctx := gc.Request.Context()

	userId, err := app.CurrentUserId(gc)
	if err != nil {
		gc.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	if userId < 0 {
		gc.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Parse organizer ID from path param
	idStr := gc.Param("id")
	organizerId, err := strconv.Atoi(idStr)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "invalid organizer id"})
		return
	}

	sql := app.Singleton.SqlAdminChoosableUserEventOrganizers

	rows, err := db.Query(ctx, sql, userId, organizerId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type UserEventOrganizer struct {
		Id          int64  `json:"id"`
		Name        string `json:"name"`
		City        string `json:"city"`
		CountryCode string `json:"country_code"`
	}

	var organizers []UserEventOrganizer

	for rows.Next() {
		var ueo UserEventOrganizer
		if err := rows.Scan(
			&ueo.Id,
			&ueo.Name,
			&ueo.City,
			&ueo.CountryCode,
		); err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		organizers = append(organizers, ueo)
	}

	if err := rows.Err(); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(organizers) == 0 {
		// It's better to return an empty array instead of 204 so clients can safely parse it.
		gc.JSON(http.StatusOK, []UserEventOrganizer{})
		return
	}

	gc.JSON(http.StatusOK, organizers)
}
