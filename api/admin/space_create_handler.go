package api_admin

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

func SpaceCreateHandler(gc *gin.Context) {
	pool := app.Singleton.MainDbPool

	type UpdateRequest struct {
		VenueId         int     `json:"venue_id"`
		Name            *string `json:"name"`
		SpaceTypeId     int     `json:"space_type_id"`
		BuildingLevel   int     `json:"building_level"`
		TotalCapacity   int     `json:"total_capacity"`
		SeatingCapacity int     `json:"seating_capacity"`
		WebsiteUrl      *string `json:"website_url"`
	}

	// TODO: Check permissions by user and OrganizerId

	var req UpdateRequest
	if err := gc.ShouldBindJSON(&req); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Begin transaction
	tx, err := pool.Begin(gc)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
		return
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(gc)
		}
	}()

	var newId int
	insertSpaceQuery := `
		INSERT INTO {{schema}}.venue
			(venue_id, name, space_type_id, building_level, total_capacity, seating_capacity, website_url)
		VALUES
			($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`
	insertSpaceQuery = strings.Replace(insertSpaceQuery, "{{schema}}", app.Singleton.Config.DbSchema, 1)

	err = tx.QueryRow(gc, insertSpaceQuery,
		req.VenueId,
		req.Name,
		req.SpaceTypeId,
		req.BuildingLevel,
		req.TotalCapacity,
		req.SeatingCapacity,
		req.WebsiteUrl,
	).Scan(&newId)

	if err != nil {
		_ = tx.Rollback(gc)
		gc.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("insert space failed: %v", err)})
		return
	}

	// Commit transaction
	if err = tx.Commit(gc); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit transaction"})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"id":      newId,
		"message": "Space created successfully",
	})
}
