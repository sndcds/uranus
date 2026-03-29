package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
)

func (h *ApiHandler) AdminUpdateEventVenue(gc *gin.Context) {
	ctx := gc.Request.Context()
	apiRequest := grains_api.NewRequest(gc, "admin-update-event-venue")

	eventUuid := gc.Param("eventUuid")
	if eventUuid == "" {
		apiRequest.Error(http.StatusBadRequest, "eventUuid is required")
		return
	}

	var payload struct {
		VenueUuid    *string `json:"venue_uuid"`
		SpaceUuid    *string `json:"space_uuid"`
		MeetingPoint *string `json:"meeting_point"`
		OnlineLink   *string `json:"online_link"`
	}

	if err := gc.ShouldBindJSON(&payload); err != nil {
		apiRequest.PayloadError()
		return
	}

	setClauses := []string{}
	args := []interface{}{}
	argPos := 1

	if payload.VenueUuid != nil {
		setClauses = append(setClauses, fmt.Sprintf("venue_id = $%d", argPos))
		args = append(args, *payload.VenueUuid)
		argPos++
	}

	if payload.SpaceUuid != nil {
		setClauses = append(setClauses, fmt.Sprintf("space_id = $%d", argPos))
		args = append(args, *payload.SpaceUuid) // Actual number
		argPos++
	} else {
		// Explicitly set NULL
		setClauses = append(setClauses, fmt.Sprintf("space_id = $%d", argPos))
		args = append(args, nil)
		argPos++
	}

	if payload.MeetingPoint != nil {
		setClauses = append(setClauses, fmt.Sprintf("meeting_point = $%d", argPos))
		args = append(args, *payload.MeetingPoint)
		argPos++
	}

	if payload.OnlineLink != nil {
		setClauses = append(setClauses, fmt.Sprintf("online_link = $%d", argPos))
		args = append(args, *payload.OnlineLink)
		argPos++
	}

	if len(setClauses) == 0 {
		apiRequest.Error(http.StatusBadRequest, "no fields to update")
		return
	}

	query := fmt.Sprintf(`UPDATE %s.event SET %s WHERE id = $%d`,
		h.DbSchema,
		strings.Join(setClauses, ", "),
		argPos, // Last placeholder is for WHERE id
	)

	args = append(args, eventUuid) // eventId is the last parameter

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		// Handle venue/space update
		// Check if the space belongs to the venue
		if payload.SpaceUuid != nil && payload.VenueUuid != nil {
			spaceExists := false
			if payload.SpaceUuid != nil {
				query := fmt.Sprintf(`SELECT EXISTS(SELECT 1 FROM %s.space WHERE id=$1 AND venue_id=$2)`, h.DbSchema)
				if err := tx.QueryRow(ctx, query, *payload.SpaceUuid, *payload.VenueUuid).Scan(&spaceExists); err != nil {
					return &ApiTxError{
						Code: http.StatusInternalServerError,
						Err:  errors.New("space doesnt belong to a venue"),
					}
				}

				if !spaceExists {
					return &ApiTxError{
						Code: http.StatusBadRequest,
						Err:  fmt.Errorf("space %d does not belong to venue %d", *payload.SpaceUuid, *payload.VenueUuid),
					}
				}
			}
		} else if payload.SpaceUuid != nil && payload.VenueUuid == nil {
			return &ApiTxError{
				Code: http.StatusBadRequest,
				Err:  errors.New("cannot update space without venueId"),
			}
		}

		res, err := tx.Exec(ctx, query, args...)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("failed to update event: %v", err),
			}
		}

		if res.RowsAffected() == 0 {
			return &ApiTxError{
				Code: http.StatusNotFound,
				Err:  errors.New("event not found"),
			}
		}

		err = RefreshEventProjections(ctx, tx, "event", []string{eventUuid})
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("refresh projection tables failed: %v", err),
			}
		}

		return nil
	})
	if txErr != nil {
		apiRequest.DatabaseError()
		return
	}

	apiRequest.SuccessNoData(http.StatusOK, "")
}
