package api_admin

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

func CreateVenueHandler(gc *gin.Context) {
	pool := app.Singleton.MainDbPool

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

	var req UpdateRequest
	if err := gc.ShouldBindJSON(&req); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate WKT point
	wktPoint, err := app.GenerateWKT(req.Latitude, req.Longitude)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Begin transaction
	tx, err := pool.Begin(gc)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
		return
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(gc)
		}
	}()

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
	insertVenueQuery = strings.Replace(insertVenueQuery, "{{schema}}", app.Singleton.Config.DbSchema, 1)

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
		_ = tx.Rollback(gc)
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
