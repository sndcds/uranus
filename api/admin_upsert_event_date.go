package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type EventDateLocationInput struct {
	Name        *string  `json:"name"`
	Street      *string  `json:"street"`
	HouseNumber *string  `json:"house_number"`
	PostalCode  *string  `json:"postal_code"`
	City        *string  `json:"city"`
	CountryCode *string  `json:"country_code"`
	StateCode   *string  `json:"state_code"`
	Latitude    *float64 `json:"lat"`
	Longitude   *float64 `json:"lon"`
	Description *string  `json:"description"`
}

type EventDateInput struct {
	StartDate string                 `json:"start_date" binding:"required"`
	StartTime string                 `json:"start_time" binding:"required"`
	EndDate   *string                `json:"end_date"`
	EndTime   *string                `json:"end_time"`
	EntryTime *string                `json:"entry_time"`
	AllDay    bool                   `json:"all_day"`
	VenueId   *int                   `json:"venue_id"`
	SpaceId   *int                   `json:"space_id"`
	Location  EventDateLocationInput `json:"location"`
}

func (h *ApiHandler) AdminUpsertEventDate(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := h.DbPool
	userId := gc.GetInt("user-id")

	eventId, ok := ParamInt(gc, "eventId")
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "event ID is required"})
		return
	}

	dateId, ok := ParamInt(gc, "dateId")
	fmt.Println("dateId:", dateId)
	if !ok {
		dateId = -1 // New event date must be inserted
	}
	fmt.Println("dateId 2:", dateId)

	fmt.Println("userId:", userId)
	fmt.Println("eventId:", eventId)

	// TODO: Check Permissions!

	var incoming EventDateInput
	if err := gc.ShouldBindJSON(&incoming); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !incoming.HasVenue() && !incoming.HasLocation() {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "event date must have either venue_id or location"})
		return
	} else if incoming.HasVenue() && incoming.HasLocation() {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "event date cannot have both venue_id and location"})
		return
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var locationId *int
	if incoming.HasLocation() {
		locationSql := `
			INSERT INTO {{schema}}.event_location (
				name,
				street,
				house_number,
				postal_code,
				city,
				country_code,
				state_code,
			    wkb_geometry,
				description,
				created_by
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, ST_SetSRID(ST_MakePoint($8, $9), 4326), $10, $11)
			RETURNING id`
		locationSql = strings.Replace(locationSql, "{{schema}}", h.Config.DbSchema, 1)
		location := incoming.Location
		err = tx.QueryRow(ctx, locationSql,
			location.Name,
			location.Street,
			location.HouseNumber,
			location.PostalCode,
			location.City,
			location.CountryCode,
			location.StateCode,
			location.Longitude,
			location.Latitude,
			location.Description,
			userId,
		).Scan(&locationId)
		if err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to insert event location: %v", err)})
			return
		}
	}

	// Compute entry_time
	var entryTime interface{}
	if incoming.EntryTime != nil && *incoming.EntryTime != "" {
		entryTime = *incoming.EntryTime
	} else {
		entryTime = nil
	}

	if dateId < 0 {
		fmt.Println("insert new date")
		// Insert new event date
		insertSql := fmt.Sprintf(`
			INSERT INTO %s.event_date 
				(event_id, venue_id, space_id, location_id, start_date, start_time, end_date, end_time, entry_time, all_day, created_by)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			RETURNING id
		`, h.Config.DbSchema)

		var newEventDateId int
		err = tx.QueryRow(ctx, insertSql,
			eventId,
			incoming.VenueId,
			incoming.SpaceId,
			locationId,
			incoming.StartDate,
			incoming.StartTime,
			incoming.EndDate,
			incoming.EndTime,
			entryTime,
			incoming.AllDay,
			userId,
		).Scan(&newEventDateId)
		if err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to insert event date: %v", err)})
			return
		}

		if err = tx.Commit(ctx); err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to commit transaction: %v", err)})
			return
		}

		gc.JSON(http.StatusOK, gin.H{
			"event_id":      eventId,
			"event_date_id": newEventDateId,
			"message":       "event date created successfully",
		})
		return
	}

	// Update existing event date
	sqlUpdate := fmt.Sprintf(`
		UPDATE %s.event_date
		SET venue_id = $1, space_id = $2, start_date = $3, start_time = $4, end_date = $5, end_time = $6, entry_time = $7, all_day = $8
		WHERE event_id = $9 AND id = $10 
	`, h.Config.DbSchema)

	cmdTag, err := tx.Exec(ctx, sqlUpdate,
		incoming.VenueId,
		incoming.SpaceId,
		incoming.StartDate,
		incoming.StartTime,
		incoming.EndDate,
		incoming.EndTime,
		entryTime,
		incoming.AllDay,
		eventId,
		dateId,
	)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to update event date: %v", err)})
		return
	}

	if cmdTag.RowsAffected() == 0 {
		gc.JSON(http.StatusNotFound, gin.H{"error": "event date not found"})
		return
	}

	if err = tx.Commit(ctx); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to commit transaction: %v", err)})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"event_id":      eventId,
		"event_date_id": dateId,
		"message":       "event date updated successfully",
	})
}

func (e *EventDateInput) HasVenue() bool {
	return e.VenueId != nil
}

func (e *EventDateInput) HasLocation() bool {
	l := e.Location
	return l.Name != nil ||
		l.Street != nil ||
		l.HouseNumber != nil ||
		l.PostalCode != nil ||
		l.City != nil ||
		l.CountryCode != nil ||
		l.StateCode != nil ||
		l.Latitude != nil ||
		l.Longitude != nil ||
		l.Description != nil
}
