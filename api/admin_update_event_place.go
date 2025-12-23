package api

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

func (h *ApiHandler) AdminUpdateEventPlace(gc *gin.Context) {
	ctx := gc.Request.Context()

	eventId, ok := ParamInt(gc, "eventId")
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "event Id is required"})
		return
	}

	var req struct {
		VenueId             *int     `json:"venue_id"`
		SpaceId             *int     `json:"space_id"`
		LocationName        *string  `json:"location_name"`
		LocationStreet      *string  `json:"location_street"`
		LocationHouseNumber *string  `json:"location_house_number"`
		LocationPostalCode  *string  `json:"location_postal_code"`
		LocationCity        *string  `json:"location_city"`
		LocationLat         *float64 `json:"location_latitude"`
		LocationLon         *float64 `json:"location_longitude"`
	}
	if err := gc.ShouldBindJSON(&req); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	usingVenue := req.VenueId != nil
	usingLocation := req.LocationName != nil && *req.LocationName != ""

	if !(usingVenue || usingLocation) {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "either venue/space or location must be provided"})
		return
	}
	if usingVenue && usingLocation {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "provide either venue/space or location, not both"})
		return
	}

	var newLocationId int64

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		if usingVenue {
			// Handle venue/space update
			// Check if the space belongs to the venue
			var setSpaceId *int
			spaceExists := false
			if req.SpaceId != nil {
				query := fmt.Sprintf(`SELECT EXISTS(SELECT 1 FROM %s.space WHERE id=$1 AND venue_id=$2)`, h.DbSchema)
				if err := tx.QueryRow(ctx, query, *req.SpaceId, *req.VenueId).Scan(&spaceExists); err != nil {
					return &ApiTxError{
						Code: http.StatusInternalServerError,
						Err:  fmt.Errorf("failed to check space: %v", err),
					}
				}
			}

			if spaceExists {
				setSpaceId = req.SpaceId
			}

			var prevLocationId sql.NullInt64
			query := fmt.Sprintf(`SELECT location_id FROM %s.event WHERE id = $1`, h.DbSchema)
			err := tx.QueryRow(ctx, query, eventId).Scan(&prevLocationId)
			if err != nil {
				if err == pgx.ErrNoRows {
					// handle no rows found if needed
					prevLocationId.Valid = false
				} else {
					return &ApiTxError{
						Code: http.StatusInternalServerError,
						Err:  fmt.Errorf("failed to get location Id: %v", err),
					}
				}
			}

			query = fmt.Sprintf(`UPDATE %s.event SET venue_id = $1, space_id = $2, location_id = NULL WHERE id = $3`, h.DbSchema)
			_, err = tx.Exec(ctx, query, *req.VenueId, setSpaceId, eventId)
			if err != nil {
				return &ApiTxError{
					Code: http.StatusInternalServerError,
					Err:  fmt.Errorf("failed to update event: %v", err),
				}
			}

			if prevLocationId.Valid {
				query = fmt.Sprintf(`DELETE FROM %s.event_location WHERE id = $1`, h.DbSchema)
				_, err = tx.Exec(ctx, query, prevLocationId.Int64)
				if err != nil {
					return &ApiTxError{
						Code: http.StatusInternalServerError,
						Err:  fmt.Errorf("failed to delete event location: %v", err),
					}
				}
			}
		} else if usingLocation {
			// Handle custom location update
			var locationId sql.NullInt64
			query := fmt.Sprintf(`SELECT location_id FROM %s.event WHERE id = $1`, h.DbSchema)
			err := tx.QueryRow(ctx, query, eventId).Scan(&locationId)
			if err != nil {
				if err == pgx.ErrNoRows {
					return &ApiTxError{
						Code: http.StatusNotFound,
						Err:  fmt.Errorf("event not found"),
					}
				}
				return &ApiTxError{
					Code: http.StatusNotFound,
					Err:  fmt.Errorf("failed to get event location: %v", err),
				}
			}

			if locationId.Valid {
				query = fmt.Sprintf(`DELETE FROM %s.event_location WHERE id = $1`, h.DbSchema)
				_, err = tx.Exec(ctx, query, locationId.Int64)
				if err != nil {
					return &ApiTxError{
						Code: http.StatusNotFound,
						Err:  fmt.Errorf("failed to delete event location: %v", err),
					}
				}
			}

			query = fmt.Sprintf(`
INSERT INTO %s.event_location (name, street, house_number, postal_code, city, wkb_pos)
VALUES ($1, $2, $3, $4, $5, ST_SetSRID(ST_MakePoint($6, $7), 4326)) RETURNING id`,
				h.DbSchema)

			err = tx.QueryRow(ctx, query,
				req.LocationName,
				req.LocationStreet,
				req.LocationHouseNumber,
				req.LocationPostalCode,
				req.LocationCity,
				req.LocationLon,
				req.LocationLat,
			).Scan(&newLocationId)
			if err != nil {
				return &ApiTxError{
					Code: http.StatusNotFound,
					Err:  fmt.Errorf("failed to insert new event location: %v", err),
				}
			}

			// Update the event with the new location_id
			updateEventQuery := fmt.Sprintf(`UPDATE %s.event SET venue_id = NULL, space_id = NULL, location_id = $1 WHERE id = $2`, h.DbSchema)
			_, err = tx.Exec(ctx, updateEventQuery, newLocationId, eventId)
			if err != nil {
				return &ApiTxError{
					Code: http.StatusNotFound,
					Err:  fmt.Errorf("failed to update event with new location: %v", err),
				}
			}

		}

		err := RefreshEventProjections(ctx, tx, "event", []int{eventId})
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

	if newLocationId != 0 {
		gc.JSON(http.StatusOK, gin.H{
			"message":     "event location created",
			"location_id": newLocationId,
		})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"message":  "event location updated successfully",
		"event_id": eventId,
	})
}
