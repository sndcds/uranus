package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

// TODO: Review code

func (h *ApiHandler) GetChoosableVenueSpaces(gc *gin.Context) {
	ctx := gc.Request.Context()

	venueIdStr := gc.Param("venueId")
	venueId, err := strconv.Atoi(venueIdStr)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := app.UranusInstance.SqlChoosableVenueSpaces
	rows, err := h.DbPool.Query(ctx, query, venueId)
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
