package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/uranus/app"
)

type upsertSpaceReq struct {
	SpaceUuid            *string `json:"space_uuid"`
	VenueUuid            *string `json:"venue_uuid"`
	Name                 *string `json:"name"`
	Description          *string `json:"description"`
	SpaceTypeId          *int    `json:"space_type_id"`
	BuildingLevel        *int    `json:"building_level"`
	TotalCapacity        *int    `json:"total_capacity"`
	SeatingCapacity      *int    `json:"seating_capacity"`
	WebLink              *string `json:"web_link"`
	AccessibilityFlags   *string `json:"accessibility_flags"` // Comes as string, as 64 bit int is not supported in JSON
	AccessibilitySummary *string `json:"accessibility_summary"`
}

func (h *ApiHandler) AdminUpsertSpace(gc *gin.Context) {
	ctx := gc.Request.Context()

	var req upsertSpaceReq
	if err := gc.ShouldBindJSON(&req); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.SpaceUuid == nil && req.VenueUuid == nil {
		gc.JSON(
			http.StatusBadRequest,
			gin.H{"error": "venueId is required when creating a space"},
		)
		return
	}

	var spaceUuid string

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {

		if req.SpaceUuid == nil {
			// Create
			err := tx.QueryRow(
				ctx,
				app.UranusInstance.SqlInsertSpace,
				req.VenueUuid,
				req.Name,
				req.Description,
				req.SpaceTypeId,
				req.BuildingLevel,
				req.TotalCapacity,
				req.SeatingCapacity,
				req.WebLink,
				req.AccessibilityFlags,
				req.AccessibilitySummary,
			).Scan(&spaceUuid)

			if err != nil {
				return &ApiTxError{
					Code: http.StatusBadRequest,
					Err:  fmt.Errorf("insert space failed: %w", err),
				}
			}

		} else {
			// Update
			spaceUuid = *req.SpaceUuid

			_, err := tx.Exec(
				ctx,
				app.UranusInstance.SqlUpdateSpace,
				spaceUuid,
				req.Name,
				req.Description,
				req.SpaceTypeId,
				req.BuildingLevel,
				req.TotalCapacity,
				req.SeatingCapacity,
				req.WebLink,
				req.AccessibilityFlags,
				req.AccessibilitySummary,
			)

			if err != nil {
				return &ApiTxError{
					Code: http.StatusInternalServerError,
					Err:  fmt.Errorf("update space failed: %w", err),
				}
			}
		}

		if err := RefreshEventProjections(ctx, tx, "space", []string{spaceUuid}); err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("refresh projection tables failed: %w", err),
			}
		}

		return nil
	})

	if txErr != nil {
		gc.JSON(txErr.Code, gin.H{"error": txErr.Error()})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"message": "Space upserted successfully",
		"uuid":    spaceUuid,
	})
}
