package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

// PermissionNote: User must be authenticated.
// The SQL query already filters events so that only events belonging to organizations
// the authenticated user is linked to via `user_organization_link` are returned.
// No additional permission checks are required in Go for access to the event list.
// Purpose: Returns the dashboard list of events for a given organization.
// PermissionChecks: Handled entirely by the SQL query.
// Verified: 2026-01-12, Roald

func (h *ApiHandler) AdminGetOrganizationEvents(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := h.userId(gc)

	organizationId, ok := ParamInt(gc, "organizationId")
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization id"})
		return
	}

	var events []model.EventDahboardEntry
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

		rows, err := tx.Query(ctx, app.UranusInstance.SqlAdminGetOrganizationEvents, organizationId, startDate, userId)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("Internal server error: %v", err),
			}
		}
		defer rows.Close()

		var eventTypesData []byte

		for rows.Next() {
			var e model.EventDahboardEntry
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
					Err:  fmt.Errorf("Internal server error: %v", err),
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
				Err:  fmt.Errorf("Internal server error: %v", err),
			}
		}

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
			"events":        []model.EventDahboardEntry{},
		})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"can_add_event": canAddEvent,
		"events":        events,
	})
}
