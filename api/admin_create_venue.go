package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

// TODO: Review code

func (h *ApiHandler) AdminCreateVenue(gc *gin.Context) {
	ctx := gc.Request.Context()
	db := h.DbPool

	type UpdateRequest struct {
		OrganizerId  int     `json:"organizer_id"`
		Name         *string `json:"name"`
		Street       *string `json:"street"`
		HouseNumber  *string `json:"house_number"`
		PostalCode   *string `json:"postal_code"`
		City         *string `json:"city"`
		CountryCode  *string `json:"country_code"`
		StateCode    *string `json:"state_code"`
		ContactEmail *string `json:"contact_email"`
		WebsiteUrl   *string `json:"website_url"`
		ContactPhone *string `json:"contact_phone"`
		Latitude     float64 `json:"latitude"`
		Longitude    float64 `json:"longitude"`
	}

	// TODO: Check permissions by user and OrganizerId

	// Read body so we can get detailed errors
	body, err := io.ReadAll(gc.Request.Body)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	var req UpdateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		// Detailed error handling
		var ute *json.UnmarshalTypeError
		switch {
		case errors.As(err, &ute):
			// ute.Field may be empty in some Go versions; try to extract a friendly name
			field := ute.Field
			if field == "" {
				// try to fall back to the JSON field name from ute.Struct? (may be empty)
				field = ute.Field
			}
			gc.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("invalid type for field %q: expected %v but got %v", field, ute.Type, ute.Value),
				"hint":  "latitude and longitude must be numbers (float), e.g. 52.520008",
			})
			return
		case err != nil:
			// generic json error
			gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	// Generate WKT point
	wktPoint, err := app.GenerateWKT(req.Latitude, req.Longitude)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Begin transaction
	tx, err := db.Begin(gc)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Insert organizer
	var newId int
	insertVenueQuery := `
		INSERT INTO {{schema}}.venue
			(organizer_id,
			 name,
			 street,
			 house_number,
			 postal_code,
			 city, 
			 country_code,
			 state_code,
			 contact_email,
			 contact_phone,
			 website_url,
			 wkb_geometry)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, ST_GeomFromText($12, 4326))
		RETURNING id
	`
	insertVenueQuery = strings.Replace(insertVenueQuery, "{{schema}}", h.Config.DbSchema, 1)

	err = tx.QueryRow(gc, insertVenueQuery,
		req.OrganizerId,
		req.Name,
		req.Street,
		req.HouseNumber,
		req.PostalCode,
		req.City,
		req.CountryCode,
		req.StateCode,
		req.ContactEmail,
		req.ContactPhone,
		req.WebsiteUrl,
		wktPoint,
	).Scan(&newId)

	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("insert venue failed: %v", err)})
		return
	}

	// TODO: Insert Website URL

	// Commit transaction
	if err = tx.Commit(gc); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit transaction"})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"id":      newId,
		"message": "Venue created successfully",
	})
}
