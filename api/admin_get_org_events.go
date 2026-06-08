package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
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

func (h *ApiHandler) AdminGetOrgEvents(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "get-user-org-event-list")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	orgUuid := gc.Param("orgUuid")
	if orgUuid == "" {
		apiRequest.Error(http.StatusBadRequest, "invalid orgUuid")
		return
	}

	var events []model.AdminListEvent
	var orgPermissions app.Permissions

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		var err error

		/*
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
		*/

		rows, err := tx.Query(ctx, app.UranusInstance.SqlAdminGetOrgEvents, userUuid, orgUuid)
		if err != nil {
			debugf(err.Error())
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("Internal server error: %v", err),
			}
		}
		defer rows.Close()

		var eventTypesData []byte

		for rows.Next() {
			var e model.AdminListEvent
			err := rows.Scan(
				&e.Uuid,
				&e.DateUuid,
				&e.ReleaseStatus,
				&e.ReleaseDate,
				&e.Categories,
				&eventTypesData,
				&e.Title,
				&e.Subtitle,
				&e.ImageUuid,
				&e.ImageUrl,
				&e.OnlineLink,
				&e.OrgUuid,
				&e.OrgName,
				&e.OrgCity,
				&e.VenueUuid,
				&e.VenueName,
				&e.VenueCity,
				&e.SpaceUuid,
				&e.SpaceName,

				&e.StartDate,
				&e.StartTime,
				&e.EndDate,
				&e.EndTime,

				&e.CanEditEvent,
				&e.CanDeleteEvent,
				&e.CanReleaseEvent,
				&e.CanViewEventInsights,
				&e.SeriesTotal,
				&e.SeriesIndex,
				&e.UpcomingDatesCount,
				&e.NextDate,
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

		orgPermissions, err = h.GetUserOrgPermissionsTx(gc, tx, userUuid, orgUuid)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("Internal server error: %v", err),
			}
		}

		return nil
	})

	if txErr != nil {
		apiRequest.Error(txErr.Code, txErr.Error())
		return
	}

	canAddEvent := orgPermissions.Has(app.UserPermAddEvent)
	apiRequest.SetMeta("can_add_event", canAddEvent)
	apiRequest.SetMeta("total_events", len(events))
	apiRequest.Success(http.StatusOK, events)
}
