package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// TODO: Review code

func (h *ApiHandler) AdminCreateSpace(gc *gin.Context) {
	ctx := gc.Request.Context()

	type UpdateRequest struct {
		VenueId              int     `json:"venue_id"`
		Name                 *string `json:"name"`
		Description          *string `json:"description"`
		SpaceTypeId          int     `json:"space_type_id"`
		BuildingLevel        int     `json:"building_level"`
		TotalCapacity        int     `json:"total_capacity"`
		SeatingCapacity      int     `json:"seating_capacity"`
		WebsiteUrl           *string `json:"website_url"`
		AccessibilityFlags   int64   `json:"accessibility_flags"`
		AccessibilitySummary *string `json:"accessibility_summary"`
	}

	// TODO: Check permissions by user and OrganizationId

	var req UpdateRequest
	if err := gc.ShouldBindJSON(&req); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Begin transaction
	tx, err := h.DbPool.Begin(gc)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var newId int
	insertSpaceQuery := `
		INSERT INTO {{schema}}.space
			(venue_id, name, description, space_type_id, building_level, total_capacity, seating_capacity, website_url, accessibility_flags, accessibility_summary)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`
	insertSpaceQuery = strings.Replace(insertSpaceQuery, "{{schema}}", h.Config.DbSchema, 1)

	err = tx.QueryRow(gc, insertSpaceQuery,
		req.VenueId,
		req.Name,
		req.Description,
		req.SpaceTypeId,
		req.BuildingLevel,
		req.TotalCapacity,
		req.SeatingCapacity,
		req.WebsiteUrl,
		req.AccessibilityFlags,
		req.AccessibilitySummary,
	).Scan(&newId)

	if err != nil {
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
