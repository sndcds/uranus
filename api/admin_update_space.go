package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/uranus/app"
)

type spaceReq struct {
	VenueID              *int    `json:"venue_id"`
	Name                 *string `json:"name"`
	Description          *string `json:"description"`
	TotalCapacity        *int    `json:"total_capacity"`
	SeatingCapacity      *int    `json:"seating_capacity"`
	SpaceTypeID          *int    `json:"space_type_id"`
	BuildingLevel        *int    `json:"building_level"`
	WebsiteUrl           *string `json:"website_url"`
	AccessibilityFlags   *int64  `json:"accessibility_flags"`
	AccessibilitySummary *string `json:"accessibility_summary"`
}

func (h *ApiHandler) AdminUpdateSpace(gc *gin.Context) {
	ctx := gc.Request.Context()

	spaceId, ok := ParamInt(gc, "spaceId")
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "Space Id is required"})
		return
	}

	var req spaceReq
	if err := gc.ShouldBindJSON(&req); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		_, err := h.DbPool.Exec(
			ctx,
			app.Singleton.SqlUpdateSpace,
			spaceId,
			req.Name,
			req.Description,
			req.SpaceTypeID,
			req.BuildingLevel,
			req.TotalCapacity,
			req.SeatingCapacity,
			req.WebsiteUrl,
			req.AccessibilityFlags,
			req.AccessibilitySummary,
		)

		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("failed to update event space: %v", err),
			}
		}

		err = RefreshEventProjections(ctx, tx, "space", []int{spaceId})
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("refresh projection tables failed: %v", err),
			}
		}

		return nil
	})
	if txErr != nil {
		gc.JSON(txErr.Code, gin.H{"error": txErr.Error()})
		return
	}

	gc.JSON(http.StatusOK, gin.H{"message": "Space updated successfully"})
}
