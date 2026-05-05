package api

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

// PermissionNote: User must be authenticated.
// The endpoint returns venue details only if the authenticated user
// is linked to the venue (via the SQL query).
// PermissionChecks: Already enforced in SQL; no additional checks needed in Go.
// Verified: 2026-01-12, Roald

func (h *ApiHandler) AdminGetVenue(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-get-venue")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	venueUuid := gc.Param("venueUuid")
	if venueUuid == "" {
		apiRequest.Required("venueUuid is required")
		return
	}

	var venue model.Venue
	var imagesRaw []byte
	query := app.UranusInstance.SqlAdminGetVenue
	row := h.DbPool.QueryRow(ctx, query, venueUuid, userUuid)

	err := row.Scan(
		&venue.Uuid,
		&venue.Name,
		&venue.Description,
		&venue.Type,
		&venue.OrgUuid,
		&venue.OpenedAt,
		&venue.ClosedAt,
		&venue.ContactEmail,
		&venue.ContactPhone,
		&venue.WebLink,
		&venue.Street,
		&venue.HouseNumber,
		&venue.PostalCode,
		&venue.City,
		&venue.State,
		&venue.Country,
		&venue.Lon,
		&venue.Lat,
		&imagesRaw,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			apiRequest.Error(http.StatusNotFound, "venue not found")
			return
		}
		debugf(err.Error())
		apiRequest.DatabaseError()
		return
	}

	if len(imagesRaw) > 0 {
		if err := json.Unmarshal(imagesRaw, &venue.Images); err != nil {
			debugf(err.Error())
			apiRequest.DatabaseError()
			return
		}
	} else {
		venue.Images = make(map[string]model.Image)
	}

	apiRequest.Success(http.StatusOK, venue, "venue loaded successfully")
}
