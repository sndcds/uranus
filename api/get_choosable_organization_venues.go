package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

// TODO: Review code

func (h *ApiHandler) GetChoosableOrganizationVenues(gc *gin.Context) {
	db := app.Singleton.MainDbPool
	ctx := gc.Request.Context()

	organizationIdStr := gc.Param("organizationId")
	organizationId, err := strconv.Atoi(organizationIdStr)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sql := app.Singleton.SqlChoosableOrganizationVenues
	rows, err := db.Query(ctx, sql, organizationId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type Venue struct {
		Id   int64   `json:"id"`
		Name *string `json:"name"`
	}

	var venues []Venue

	for rows.Next() {
		var venue Venue
		if err := rows.Scan(
			&venue.Id,
			&venue.Name,
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
		gc.JSON(http.StatusOK, []Venue{})
		return
	}

	gc.JSON(http.StatusOK, venues)
}
