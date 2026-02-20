package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// TODO: Review code

func (h *ApiHandler) GetChoosableVenues(gc *gin.Context) {
	ctx := gc.Request.Context()

	query := fmt.Sprintf("SELECT id, name FROM %s.venue ORDER BY LOWER(name)", h.DbSchema)
	rows, err := h.DbPool.Query(ctx, query)
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
