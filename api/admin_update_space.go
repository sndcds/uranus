package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

type SpaceUpdateRequest struct {
	VenueID              *int    `json:"venue_id"`
	Name                 *string `json:"name"`
	TotalCapacity        *int    `json:"total_capacity"`
	SeatingCapacity      *int    `json:"seating_capacity"`
	SpaceTypeID          *int    `json:"space_type_id"`
	BuildingLevel        *int    `json:"building_level"`
	WebsiteURL           *string `json:"website_url"`
	AccessibilitySummary *string `json:"accessibility_summary"`
	AccessibilityFlags   *int64  `json:"accessibility_flags"`
	Description          *string `json:"description"`
}

func (h *ApiHandler) AdminUpdateSpace(gc *gin.Context) {
	pool := app.Singleton.MainDbPool
	ctx := gc.Request.Context()

	spaceId := gc.Param("spaceId")
	if spaceId == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "Space ID is required"})
		return
	}

	var req SpaceUpdateRequest
	if err := gc.ShouldBindJSON(&req); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := pool.Exec(
		ctx,
		app.Singleton.SqlUpdateSpace,
		spaceId,
		req.Name,
		req.Description,
		req.SpaceTypeID,
		req.BuildingLevel,
		req.TotalCapacity,
		req.SeatingCapacity,
		req.WebsiteURL,
		req.AccessibilityFlags,
		req.AccessibilitySummary,
	)

	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	gc.JSON(http.StatusOK, gin.H{"message": "Space updated successfully"})
}
