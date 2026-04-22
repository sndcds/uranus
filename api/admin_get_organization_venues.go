package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
)

// PermissionNote: User must be authenticated.
// Only returns venues for the organization if the authenticated user is linked via `user_organization_link`.
// If the user is not linked, returns HTTP 403 Forbidden.
// PermissionChecks: Already enforced in SQL.
// Verified: 2026-01-12, Roald

func (h *ApiHandler) AdminGetOrganizationVenues(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-get-organization-venues")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	type SpaceInfo struct {
		SpaceUuid      string `json:"space_uuid"`
		SpaceName      string `json:"space_name"`
		EventCount     int    `json:"event_count"`
		CanEditSpace   bool   `json:"can_edit_space"`
		CanDeleteSpace bool   `json:"can_delete_space"`
	}

	type VenueInfo struct {
		VenueUuid          *string     `json:"venue_uuid"`
		VenueName          *string     `json:"venue_name"`
		EventCount         int         `json:"event_count"`
		CanEditVenue       bool        `json:"can_edit_venue"`
		CanDeleteVenue     bool        `json:"can_delete_venue"`
		CanAddSpace        bool        `json:"can_add_space"`
		MainLogoUuid       *string     `json:"main_logo_uuid"`
		LightThemeLogoUuid *string     `json:"light_theme_logo_uuid"`
		DarkThemeLogoUuid  *string     `json:"dark_theme_logo_uuid"`
		Spaces             []SpaceInfo `json:"spaces"`
	}

	orgUuid := gc.Param("orgUuid")
	if orgUuid == "" {
		apiRequest.Error(http.StatusBadRequest, "invalid orgUuid")
		return
	}

	var err error

	startStr := gc.Query("start")
	var startDate time.Time
	if startStr != "" {
		startDate, err = time.Parse("2006-01-02", startStr)
		if err != nil {
			startDate = time.Now() // fallback on parse error
		}
	} else {
		startDate = time.Now() // fallback if param missing
	}

	var venues []VenueInfo
	var orgPermissions app.Permission

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {

		orgPermissions, err = h.GetUserOrganizationPermissionsTx(gc, tx, userUuid, orgUuid)
		if err != nil {
			return ApiErrInternal("%v", err)
		}

		rows, err := tx.Query(ctx, app.UranusInstance.SqlAdminGetOrganizationVenues, orgUuid, userUuid, startDate)
		if err != nil {
			debugf(err.Error())
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  errors.New("Internal server error"),
			}
		}
		defer rows.Close()

		var spacesJSON json.RawMessage
		var venuePermissions app.Permission

		for rows.Next() {
			var venue VenueInfo
			debugf("row")

			err := rows.Scan(
				&venue.VenueUuid,
				&venue.VenueName,
				&venue.MainLogoUuid,
				&venue.DarkThemeLogoUuid,
				&venue.LightThemeLogoUuid,
				&spacesJSON,
				&venuePermissions,
			)
			if err != nil {
				debugf(err.Error())
				return &ApiTxError{
					Code: http.StatusInternalServerError,
					Err:  err,
				}
			}

			if err := json.Unmarshal(spacesJSON, &venue.Spaces); err != nil {
				return &ApiTxError{
					Code: http.StatusInternalServerError,
					Err:  err,
				}
			}

			venue.CanEditVenue = venuePermissions.Has(app.PermEditVenue)
			venue.CanDeleteVenue = venuePermissions.Has(app.PermDeleteVenue)
			venue.CanAddSpace = venuePermissions.Has(app.PermAddSpace)

			for i := range venue.Spaces {
				venue.Spaces[i].CanEditSpace = venuePermissions.Has(app.PermEditSpace)
				venue.Spaces[i].CanDeleteSpace = venuePermissions.Has(app.PermDeleteSpace)
				venue.EventCount += venue.Spaces[i].EventCount
			}

			venues = append(venues, venue)
		}

		return nil
	})
	if txErr != nil {
		debugf(txErr.Error())
		apiRequest.InternalServerError()
		return
	}

	apiRequest.Success(http.StatusOK, gin.H{
		"can_add_venue": orgPermissions.Has(app.PermAddVenue),
		"venues":        venues,
	}, "")
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
