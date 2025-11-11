package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

func (h *ApiHandler) AdminUpdateEventVenue(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := h.DbPool

	eventId := gc.Param("eventId")
	if eventId == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "event ID is required"})
		return
	}

	var req struct {
		VenueId int `json:"venue_id" binding:"required"`
		SpaceId int `json:"space_id"`
	}
	if err := gc.ShouldBindJSON(&req); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to start transaction: %v", err)})
		return
	}
	defer tx.Rollback(ctx) // safe rollback if commit fails

	// Check if the space belongs to the venue
	sql := fmt.Sprintf(`
		SELECT EXISTS(
			SELECT 1 FROM %s.space
			WHERE id = $1 AND venue_id = $2
		)`,
		h.Config.DbSchema)

	var spaceExists bool
	if req.SpaceId != 0 {
		err = tx.QueryRow(ctx, sql, req.SpaceId, req.VenueId).Scan(&spaceExists)
		if err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to check space: %v", err)})
			return
		}
	}

	fmt.Println("eventId: ", eventId)
	fmt.Println("VenueId: ", req.VenueId)
	fmt.Println("Space.Id: ", req.SpaceId)
	fmt.Println("spaceExists: ", spaceExists)
	var setSpaceId *int
	if spaceExists {
		setSpaceId = &req.SpaceId // assign pointer to value
	} else {
		setSpaceId = nil
	}
	// Update the event
	sql = fmt.Sprintf(
		`UPDATE %s.event SET venue_id = $1, space_id = $2 WHERE id = $3`,
		h.Config.DbSchema)

	_, err = tx.Exec(ctx, sql, req.VenueId, setSpaceId, eventId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to update event: %v", err)})
		return
	}

	if err := tx.Commit(ctx); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to commit transaction: %v", err)})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"event_id": eventId,
		"message":  "event venue and space updated successfully",
	})
}
