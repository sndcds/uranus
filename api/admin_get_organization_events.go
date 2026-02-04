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
	apiResponseType := "user-organization-event-list"

	organizationId, ok := ParamInt(gc, "organizationId")
	if !ok {
		JSONError(gc, apiResponseType, http.StatusBadRequest, "invalid organizationId")
		return
	}

	var events []model.AdminListEvent
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
			var e model.AdminListEvent
			err := rows.Scan(
				&e.Id,
				&e.DateId,
				&e.Title,
				&e.Subtitle,
				&e.OrganizationId,
				&e.OrganizationName,
				&e.StartDate,
				&e.StartTime,
				&e.EndDate,
				&e.EndTime,
				&e.ReleaseStatus,
				&e.ReleaseDate,
				&e.VenueId,
				&e.VenueName,
				&e.SpaceId,
				&e.SpaceName,
				&e.ImageId,
				&e.ImageUrl,
				&eventTypesData,
				&e.CanEditEvent,
				&e.CanDeleteEvent,
				&e.CanReleaseEvent,
				&e.SeriesIndex,
				&e.SeriesTotal,
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
		debugf("API %s error: %s", apiResponseType, txErr.Error())
		JSONError(gc, apiResponseType, http.StatusBadRequest, "transaction failed")
		return
	}

	canAddEvent := organizationPermissions.Has(app.PermAddEvent)
	metadata := map[string]interface{}{
		"can_add_event": canAddEvent,
		"total_events":  len(events),
	}

	JSONSuccess(gc, apiResponseType, events, metadata)
}
