package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/sql_utils"
)

func (h *ApiHandler) AdminUpdateEventVenue(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-update-event-venue")
	ctx := gc.Request.Context()

	eventUuid := gc.Param("eventUuid")
	if eventUuid == "" {
		apiRequest.Required("eventUuid is required")
		return
	}

	var payload struct {
		VenueUuid            *string `json:"venue_uuid"`
		SpaceUuid            *string `json:"space_uuid"`
		MeetingPoint         *string `json:"meeting_point"`
		OnlineLink           *string `json:"online_link"`
		RegistrationLink     *string `json:"registration_link"`
		RegistrationDeadline *string `json:"registration_deadline"`
		RegistrationEmail    *string `json:"registration_email"`
		RegistrationPhone    *string `json:"registration_phone"`
	}

	if err := gc.ShouldBindJSON(&payload); err != nil {
		debugf(err.Error())
		apiRequest.PayloadError()
		return
	}

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		if payload.SpaceUuid != nil && payload.VenueUuid == nil {
			return &ApiTxError{
				Code: http.StatusBadRequest,
				Err:  errors.New("cannot update space without venueUuid"),
			}
		}

		if payload.SpaceUuid != nil && payload.VenueUuid != nil {
			var spaceExists bool

			query := fmt.Sprintf(`
				SELECT EXISTS(
					SELECT 1 FROM %s.space
					WHERE uuid=$1::uuid AND venue_uuid=$2::uuid
				)`, h.DbSchema)

			if err := tx.QueryRow(ctx, query,
				*payload.SpaceUuid,
				*payload.VenueUuid,
			).Scan(&spaceExists); err != nil {
				return &ApiTxError{
					Code: http.StatusInternalServerError,
					Err:  errors.New("space check failed"),
				}
			}

			if !spaceExists {
				return &ApiTxError{
					Code: http.StatusBadRequest,
					Err:  fmt.Errorf("space does not belong to venue"),
				}
			}
		}

		queryBuilder := sql_utils.NewUpdate(h.DbSchema + ".event")
		queryBuilder.
			Set("venue_uuid", payload.VenueUuid).
			SetNullable("space_uuid", payload.SpaceUuid).
			Set("meeting_point", payload.MeetingPoint).
			Set("online_link", payload.OnlineLink).
			Set("registration_link", payload.RegistrationLink).
			Set("registration_email", payload.RegistrationEmail).
			Set("registration_phone", payload.RegistrationPhone).
			Set("registration_deadline", payload.RegistrationDeadline)

		query, args, err := queryBuilder.Build()
		if err != nil {
			return &ApiTxError{
				Code: http.StatusBadRequest,
				Err:  err,
			}
		}

		// Add WHERE manually
		query += fmt.Sprintf(" WHERE uuid = $%d::uuid", len(args)+1)
		args = append(args, eventUuid)

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

		if err := RefreshEventProjections(ctx, tx, "event", []string{eventUuid}); err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("refresh projection tables failed: %v", err),
			}
		}

		return nil
	})

	if txErr != nil {
		debugf(txErr.Error())
		apiRequest.DatabaseError()
		return
	}

	apiRequest.SuccessNoData(http.StatusOK, "")
}
