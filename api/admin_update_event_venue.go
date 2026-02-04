package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

func (h *ApiHandler) AdminUpdateEventVenue(gc *gin.Context) {
	ctx := gc.Request.Context()
	apiResponseType := "admin-update-event-venue"

	eventId, ok := ParamInt(gc, "eventId")
	if !ok {
		JSONError(gc, apiResponseType, http.StatusBadRequest, "eventId is required")
		return
	}

	var payload struct {
		VenueId      *int    `json:"venue_id"`
		SpaceId      *int    `json:"space_id"`
		MeetingPoint *string `json:"meeting_point"`
		OnlineLink   *string `json:"online_link"`
	}

	if err := gc.ShouldBindJSON(&payload); err != nil {
		JSONPayloadError(gc, apiResponseType)
		return
	}

	setClauses := []string{}
	args := []interface{}{}
	argPos := 1

	if payload.VenueId != nil {
		setClauses = append(setClauses, fmt.Sprintf("venue_id = $%d", argPos))
		args = append(args, *payload.VenueId)
		argPos++
	}

	if payload.SpaceId != nil {
		setClauses = append(setClauses, fmt.Sprintf("space_id = $%d", argPos))
		args = append(args, *payload.SpaceId) // actual number
		argPos++
	} else {
		// explicitly set NULL
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
		JSONError(gc, apiResponseType, http.StatusBadRequest, "no fields to update")
		return
	}

	query := fmt.Sprintf(`UPDATE %s.event SET %s WHERE id = $%d`,
		h.DbSchema,
		strings.Join(setClauses, ", "),
		argPos, // Last placeholder is for WHERE id
	)

	args = append(args, eventId) // eventId is the last parameter
	fmt.Println("query", query)
	fmt.Println("args", args)

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		// Handle venue/space update
		// Check if the space belongs to the venue
		if payload.SpaceId != nil && payload.VenueId != nil {
			spaceExists := false
			if payload.SpaceId != nil {
				query := fmt.Sprintf(`SELECT EXISTS(SELECT 1 FROM %s.space WHERE id=$1 AND venue_id=$2)`, h.DbSchema)
				if err := tx.QueryRow(ctx, query, *payload.SpaceId, *payload.VenueId).Scan(&spaceExists); err != nil {
					return &ApiTxError{
						Code: http.StatusInternalServerError,
						Err:  fmt.Errorf("space doesnt belong to a venue"),
					}
				}

				if !spaceExists {
					return &ApiTxError{
						Code: http.StatusBadRequest,
						Err:  fmt.Errorf("space %d does not belong to venue %d", *payload.SpaceId, *payload.VenueId),
					}
				}
			}
		} else if payload.SpaceId != nil && payload.VenueId == nil {
			return &ApiTxError{
				Code: http.StatusBadRequest,
				Err:  fmt.Errorf("cannot update space without venueId"),
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
				Err:  fmt.Errorf("event not found"),
			}
		}

		err = RefreshEventProjections(ctx, tx, "event", []int{eventId})
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("refresh projection tables failed: %v", err),
			}
		}

		return nil
	})
	if txErr != nil {
		JSONDatabaseError(gc, apiResponseType)
		return
	}

	JSONSuccessNoData(gc, apiResponseType)
}
