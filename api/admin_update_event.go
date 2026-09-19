package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) AdminUpdateEvent(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-update-event")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	eventUuid := gc.Param("eventUuid")
	if eventUuid == "" {
		apiRequest.Required("eventUuid is required")
		return
	}

	var payload struct {
		ReleaseStatus     NullableField[string]   `json:"release_status"`
		ReleaseDate       NullableField[string]   `json:"release_date"`
		ContentLanguage   NullableField[string]   `json:"content_language"`
		Categories        NullableField[[]int]    `json:"categories"`
		Title             NullableField[string]   `json:"title"`
		Subtitle          NullableField[string]   `json:"subtitle"`
		Description       NullableField[string]   `json:"description"`
		Summary           NullableField[string]   `json:"summary"`
		LogoMode          NullableField[int]      `json:"logo_mode"`
		Tags              NullableField[[]string] `json:"tags"`
		MaxAttendees      NullableField[int]      `json:"max_attendees"`
		MinAge            NullableField[int]      `json:"min_age"`
		MaxAge            NullableField[int]      `json:"max_age"`
		ParticipationInfo NullableField[string]   `json:"participation_info"`
		PriceType         NullableField[string]   `json:"price_type"`
		MinPrice          NullableField[float64]  `json:"min_price"`
		MaxPrice          NullableField[float64]  `json:"max_price"`
		Currency          NullableField[string]   `json:"currency"`
		TicketFlags       *[]string               `json:"ticket_flags"`
		TicketLink        NullableField[string]   `json:"ticket_link"`
		VisitorInfoFlags  NullableField[string]   `json:"visitor_info_flags"`

		VenueUuid            NullableField[string] `json:"venue_uuid"`
		SpaceUuid            NullableField[string] `json:"space_uuid"`
		MeetingPoint         NullableField[string] `json:"meeting_point"`
		OnlineLink           NullableField[string] `json:"online_link"`
		RegistrationLink     NullableField[string] `json:"registration_link"`
		RegistrationEmail    NullableField[string] `json:"registration_email"`
		RegistrationPhone    NullableField[string] `json:"registration_phone"`
		RegistrationDeadline NullableField[string] `json:"registration_deadline"`
	}

	decoder := json.NewDecoder(gc.Request.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&payload); err != nil {
		debugf(err.Error())
		apiRequest.PayloadError()
		return
	}

	var fieldCount int

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {

		// ------------------------------------------------------------
		// Get organization belonging to the event
		// ------------------------------------------------------------

		orgUuid, err := h.GetOrgUuidByEventUuidTx(gc, tx, eventUuid)
		if err != nil {
			return ApiErrInternal("%v", err)
		}

		if orgUuid == "" {
			return ApiErrInternal("internal orgUuid failed")
		}

		// ------------------------------------------------------------
		// Permission
		// ------------------------------------------------------------

		permissions, err := h.GetUserOrgPermissionsTx(
			gc,
			tx,
			userUuid,
			orgUuid,
		)
		if err != nil {
			return ApiErrInternal("%v", err)
		}

		if !permissions.Has(app.UserPermEditEvent) {
			return &ApiTxError{
				Code: http.StatusForbidden,
				Err:  errors.New("permission denied"),
			}
		}

		if payload.ReleaseStatus.Set &&
			!permissions.Has(app.UserPermReleaseEvent) {
			return &ApiTxError{
				Code: http.StatusForbidden,
				Err:  errors.New("permission denied to change release_status"),
			}
		}

		// ------------------------------------------------------------
		// Build UPDATE
		// ------------------------------------------------------------

		var setClauses []string
		var args []interface{}
		argPos := 1

		argPos = addUpdateClauseNullable(
			"release_status",
			payload.ReleaseStatus,
			&setClauses,
			&args,
			argPos,
		)

		argPos = addUpdateClauseNullable(
			"release_date",
			payload.ReleaseDate,
			&setClauses,
			&args,
			argPos,
		)

		argPos = addUpdateClauseNullable(
			"content_iso_639_1",
			payload.ContentLanguage,
			&setClauses,
			&args,
			argPos,
		)

		argPos = addUpdateClauseNullable(
			"categories",
			payload.Categories,
			&setClauses,
			&args,
			argPos,
		)

		argPos = addUpdateClauseNullable(
			"title",
			payload.Title,
			&setClauses,
			&args,
			argPos,
		)

		argPos = addUpdateClauseNullable(
			"subtitle",
			payload.Subtitle,
			&setClauses,
			&args,
			argPos,
		)

		argPos = addUpdateClauseNullable(
			"description",
			payload.Description,
			&setClauses,
			&args,
			argPos,
		)

		argPos = addUpdateClauseNullable(
			"summary",
			payload.Summary,
			&setClauses,
			&args,
			argPos,
		)

		argPos = addUpdateClauseNullable(
			"logo_mode",
			payload.LogoMode,
			&setClauses,
			&args,
			argPos,
		)

		argPos = addUpdateClauseNullable(
			"tags",
			payload.Tags,
			&setClauses,
			&args,
			argPos,
		)

		argPos = addUpdateClauseNullable(
			"max_attendees",
			payload.MaxAttendees,
			&setClauses,
			&args,
			argPos,
		)

		argPos = addUpdateClauseNullable(
			"min_age",
			payload.MinAge,
			&setClauses,
			&args,
			argPos,
		)

		argPos = addUpdateClauseNullable(
			"max_age",
			payload.MaxAge,
			&setClauses,
			&args,
			argPos,
		)

		argPos = addUpdateClauseNullable(
			"participation_info",
			payload.ParticipationInfo,
			&setClauses,
			&args,
			argPos,
		)

		argPos = addUpdateClauseNullable(
			"price_type",
			payload.PriceType,
			&setClauses,
			&args,
			argPos,
		)

		argPos = addUpdateClauseNullable(
			"min_price",
			payload.MinPrice,
			&setClauses,
			&args,
			argPos,
		)

		argPos = addUpdateClauseNullable(
			"max_price",
			payload.MaxPrice,
			&setClauses,
			&args,
			argPos,
		)

		argPos = addUpdateClauseNullable(
			"currency",
			payload.Currency,
			&setClauses,
			&args,
			argPos,
		)

		argPos = addUpdateClauseNullable(
			"ticket_link",
			payload.TicketLink,
			&setClauses,
			&args,
			argPos,
		)

		argPos = addUpdateClauseNullable(
			"visitor_info_flags",
			payload.VisitorInfoFlags,
			&setClauses,
			&args,
			argPos,
		)

		// ticket_flags is NOT NULL, therefore keep its existing
		// helper semantics: nil/empty becomes an empty array.
		argPos = addUpdateClauseStringSliceField(
			"ticket_flags",
			payload.TicketFlags,
			&setClauses,
			&args,
			argPos,
		)

		// ------------------------------------------------------------
		// Registration / location fields
		// ------------------------------------------------------------

		argPos = addUpdateClauseNullable(
			"meeting_point",
			payload.MeetingPoint,
			&setClauses,
			&args,
			argPos,
		)

		argPos = addUpdateClauseNullable(
			"online_link",
			payload.OnlineLink,
			&setClauses,
			&args,
			argPos,
		)

		argPos = addUpdateClauseNullable(
			"registration_link",
			payload.RegistrationLink,
			&setClauses,
			&args,
			argPos,
		)

		argPos = addUpdateClauseNullable(
			"registration_email",
			payload.RegistrationEmail,
			&setClauses,
			&args,
			argPos,
		)

		argPos = addUpdateClauseNullable(
			"registration_phone",
			payload.RegistrationPhone,
			&setClauses,
			&args,
			argPos,
		)

		argPos = addUpdateClauseNullable(
			"registration_deadline",
			payload.RegistrationDeadline,
			&setClauses,
			&args,
			argPos,
		)

		for i, arg := range args {
			fmt.Printf("  $%d = (%T) %#v\n", i+1, arg, debugArg(arg))
		}

		// ------------------------------------------------------------
		// Venue / space
		//
		// These two fields need special handling because a space
		// belongs to a specific venue.
		// ------------------------------------------------------------

		if payload.VenueUuid.Set || payload.SpaceUuid.Set {

			var currentVenueUuid *string
			var currentSpaceUuid *string

			query := fmt.Sprintf(`
				SELECT
					venue_uuid::text,
					space_uuid::text
				FROM %s.event
				WHERE uuid = $1::uuid
			`, h.DbSchema)

			err := tx.QueryRow(
				ctx,
				query,
				eventUuid,
			).Scan(
				&currentVenueUuid,
				&currentSpaceUuid,
			)

			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return &ApiTxError{
						Code: http.StatusNotFound,
						Err:  errors.New("event not found"),
					}
				}

				return ApiErrInternal("failed to get current event venue: %v", err)
			}

			// Determine the resulting venue.
			resultVenueUuid := currentVenueUuid

			if payload.VenueUuid.Set {
				resultVenueUuid = payload.VenueUuid.Value
			}

			// Determine the resulting space.
			resultSpaceUuid := currentSpaceUuid

			if payload.SpaceUuid.Set {
				resultSpaceUuid = payload.SpaceUuid.Value
			}

			// If the venue is explicitly changed and the space wasn't,
			// do not blindly keep a space belonging to the old venue.
			//
			// We only keep it if it is valid for the resulting venue.
			if payload.VenueUuid.Set && !payload.SpaceUuid.Set {
				if resultSpaceUuid != nil {

					if resultVenueUuid == nil {
						// Venue is being removed, therefore the space
						// must also be removed.
						resultSpaceUuid = nil
					} else {
						var spaceBelongsToVenue bool

						query := fmt.Sprintf(`
							SELECT EXISTS(
								SELECT 1
								FROM %s.space
								WHERE uuid = $1::uuid
								  AND venue_uuid = $2::uuid
							)
						`, h.DbSchema)

						if err := tx.QueryRow(
							ctx,
							query,
							*resultSpaceUuid,
							*resultVenueUuid,
						).Scan(&spaceBelongsToVenue); err != nil {
							return ApiErrInternal(
								"failed to validate space: %v",
								err,
							)
						}

						if !spaceBelongsToVenue {
							resultSpaceUuid = nil
						}
					}
				}
			}

			// A space can only exist if there is a venue.
			if resultSpaceUuid != nil && resultVenueUuid == nil {
				return &ApiTxError{
					Code: http.StatusBadRequest,
					Err: errors.New(
						"cannot assign a space without a venue",
					),
				}
			}

			// If a resulting space exists, make sure it belongs to
			// the resulting venue.
			if resultSpaceUuid != nil {

				var spaceBelongsToVenue bool

				query := fmt.Sprintf(`
					SELECT EXISTS(
						SELECT 1
						FROM %s.space
						WHERE uuid = $1::uuid
						  AND venue_uuid = $2::uuid
					)
				`, h.DbSchema)

				if err := tx.QueryRow(
					ctx,
					query,
					*resultSpaceUuid,
					*resultVenueUuid,
				).Scan(&spaceBelongsToVenue); err != nil {
					return ApiErrInternal(
						"failed to validate space: %v",
						err,
					)
				}

				if !spaceBelongsToVenue {
					return &ApiTxError{
						Code: http.StatusBadRequest,
						Err: errors.New(
							"space does not belong to venue",
						),
					}
				}
			}

			// Venue was explicitly supplied.
			if payload.VenueUuid.Set {
				argPos = addUpdateClauseNullable(
					"venue_uuid",
					payload.VenueUuid,
					&setClauses,
					&args,
					argPos,
				)
			}

			// Space was explicitly supplied.
			if payload.SpaceUuid.Set {
				argPos = addUpdateClauseNullable(
					"space_uuid",
					payload.SpaceUuid,
					&setClauses,
					&args,
					argPos,
				)
			} else if payload.VenueUuid.Set &&
				currentSpaceUuid != nil &&
				resultSpaceUuid == nil {

				// Venue changed and the old space is no longer valid.
				// Clear it automatically.
				setClauses = append(
					setClauses,
					fmt.Sprintf("space_uuid = $%d", argPos),
				)
				args = append(args, nil)
				argPos++
			}
		}

		fieldCount = len(setClauses)

		// ------------------------------------------------------------
		// Nothing to update
		// ------------------------------------------------------------

		if fieldCount == 0 {
			return nil
		}

		// ------------------------------------------------------------
		// Execute UPDATE
		// ------------------------------------------------------------

		query := fmt.Sprintf(
			`UPDATE %s.event
			 SET %s
			 WHERE uuid = $%d::uuid`,
			h.DbSchema,
			strings.Join(setClauses, ", "),
			argPos,
		)

		args = append(args, eventUuid)

		res, err := tx.Exec(
			ctx,
			query,
			args...,
		)

		if err != nil {
			return ApiErrInternal(
				"failed to update event: %v",
				err,
			)
		}

		if res.RowsAffected() == 0 {
			return &ApiTxError{
				Code: http.StatusNotFound,
				Err:  errors.New("event not found"),
			}
		}

		// ------------------------------------------------------------
		// Refresh projections
		// ------------------------------------------------------------

		if err := RefreshEventProjections(
			ctx,
			tx,
			"event",
			[]string{eventUuid},
		); err != nil {
			return ApiErrInternal(
				"failed to refresh event projections: %v",
				err,
			)
		}

		return nil
	})

	// ------------------------------------------------------------
	// Response
	// ------------------------------------------------------------

	if txErr != nil {
		debugf(txErr.Error())
		apiRequest.Error(txErr.Code, txErr.Error())
		return
	}

	apiRequest.SetMeta("event_uuid", eventUuid)
	apiRequest.SetMeta("field_count", fieldCount)

	apiRequest.SuccessNoData(
		http.StatusOK,
		"",
	)
}

func debugArg(v interface{}) interface{} {
	if v == nil {
		return nil
	}

	rv := reflect.ValueOf(v)

	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}

	return rv.Interface()
}
