package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/uranus/app"
)

// TODO: Review code

func (h *ApiHandler) AdminGetOrganizationEvents(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := gc.GetInt("user-id")
	langStr := gc.DefaultQuery("lang", "en")

	type EventType struct {
		TypeID    int     `json:"type_id"`
		TypeName  string  `json:"type_name"`
		GenreID   int     `json:"genre_id"`
		GenreName *string `json:"genre_name"`
	}

	type Event struct {
		EventId               int         `json:"event_id"`
		EventDateId           int         `json:"event_date_id"`
		EventTitle            string      `json:"event_title"`
		EventSubtitle         *string     `json:"event_subtitle"`
		EventOrganizationId   int         `json:"event_organization_id"`
		EventOrganizationName *string     `json:"event_organization_name"`
		StartDate             *string     `json:"start_date"`
		StartTime             *string     `json:"start_time"`
		EndDate               *string     `json:"end_date"`
		EndTime               *string     `json:"end_time"`
		ReleaseStatusId       *int        `json:"release_status_id"`
		ReleaseStatusName     *string     `json:"release_status_name"`
		ReleaseDate           *string     `json:"release_date"`
		VenueId               *int        `json:"venue_id"`
		VenueName             *string     `json:"venue_name"`
		SpaceId               *int        `json:"space_id,omitempty"`
		SpaceName             *string     `json:"space_name,omitempty"`
		LocationId            *int        `json:"location_id"`
		LocationName          *string     `json:"location_name"`
		ImageId               *int        `json:"image_id"`
		EventTypes            []EventType `json:"event_types"`
		CanEditEvent          bool        `json:"can_edit_event"`
		CanDeleteEvent        bool        `json:"can_delete_event"`
		CanReleaseEvent       bool        `json:"can_release_event"`
		TimeSeriesIndex       int         `json:"time_series_index"`
		TimeSeries            int         `json:"time_series"`
	}

	organizationId, ok := ParamInt(gc, "organizationId")
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization id"})
		return
	}

	var events []Event
	var organizationPermissions app.Permission

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		var err error
		startStr := gc.Query("start")
		var startDate time.Time
		if startStr != "" {
			startDate, err = time.Parse("2006-01-02", startStr)
			if err != nil {
				startDate = time.Now()
			}
		} else {
			startDate = time.Now()
		}

		rows, err := tx.Query(ctx, app.Singleton.SqlAdminGetOrganizationEvents, organizationId, startDate, langStr, userId)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("Transaction failed 1: %v", err),
			}
		}
		defer rows.Close()

		var eventTypesData []byte

		for rows.Next() {
			var e Event
			err := rows.Scan(
				&e.EventId,
				&e.EventDateId,
				&e.EventTitle,
				&e.EventSubtitle,
				&e.EventOrganizationId,
				&e.EventOrganizationName,
				&e.StartDate,
				&e.StartTime,
				&e.EndDate,
				&e.EndTime,
				&e.ReleaseStatusId,
				&e.ReleaseStatusName,
				&e.ReleaseDate,
				&e.VenueId,
				&e.VenueName,
				&e.SpaceId,
				&e.SpaceName,
				&e.LocationId,
				&e.LocationName,
				&e.ImageId,
				&eventTypesData,
				&e.CanEditEvent,
				&e.CanDeleteEvent,
				&e.CanReleaseEvent,
				&e.TimeSeriesIndex,
				&e.TimeSeries,
			)
			if err != nil {
				return &ApiTxError{
					Code: http.StatusInternalServerError,
					Err:  fmt.Errorf("Transaction failed 2: %v", err),
				}
			}
			if len(eventTypesData) > 0 {
				_ = json.Unmarshal(eventTypesData, &e.EventTypes)
			}
			events = append(events, e)
		}

		organizationPermissions, err = h.GetUserOrganizationPermissions(gc, tx, userId, organizationId)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("Transaction failed 3: %v", err),
			}
		}

		fmt.Print("organizationPermissions: ", organizationPermissions)

		return nil
	})
	if txErr != nil {
		gc.JSON(txErr.Code, gin.H{"error": txErr.Error()})
		return
	}

	canAddEvent := organizationPermissions.Has(app.PermAddEvent)

	if len(events) == 0 {
		gc.JSON(http.StatusOK, gin.H{
			"can_add_event": canAddEvent,
			"events":        []Event{},
		})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"can_add_event": canAddEvent,
		"events":        events,
	})
}
