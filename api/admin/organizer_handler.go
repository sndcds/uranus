package api_admin

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/uranus/app"
)

func OrganizerCreateHandler(gc *gin.Context) {
	pool := app.Singleton.MainDbPool

	type UpdateRequest struct {
		Name         *string `json:"name"`
		Street       *string `json:"street"`
		HouseNumber  *string `json:"house_number"`
		PostalCode   *string `json:"postal_code"`
		City         *string `json:"city"`
		ContactEmail *string `json:"contact_email"`
		WebsiteUrl   *string `json:"website_url"`
		ContactPhone *string `json:"contact_phone"`
		Latitude     float64 `json:"latitude"`
		Longitude    float64 `json:"longitude"`
	}

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

	// Extract user ID from context (depends on your auth middleware)
	userId, err := app.CurrentUserId(gc)
	if err != nil {
		gc.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
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
	insertOrganizerQuery := `
		INSERT INTO {{schema}}.organizer
			(name, street, house_number, postal_code, city, contact_email, website_url, contact_phone, wkb_geometry)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, ST_GeomFromText($9, 4326))
		RETURNING id
	`
	insertOrganizerQuery = strings.Replace(insertOrganizerQuery, "{{schema}}", app.Singleton.Config.DbSchema, 1)

	err = tx.QueryRow(gc, insertOrganizerQuery,
		req.Name,
		req.Street,
		req.HouseNumber,
		req.PostalCode,
		req.City,
		req.ContactEmail,
		req.WebsiteUrl,
		req.ContactPhone,
		wktPoint,
	).Scan(&newId)

	if err != nil {
		_ = tx.Rollback(gc)
		gc.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("insert organizer failed: %v", err)})
		return
	}

	// Insert user_organizer_link
	insertLinkQuery := `
		INSERT INTO {{schema}}.user_organizer_links (user_id, organizer_id, user_role_id)
		VALUES ($1, $2, $3)
	`
	insertLinkQuery = strings.Replace(insertLinkQuery, "{{schema}}", app.Singleton.Config.DbSchema, 1)

	_, err = tx.Exec(gc, insertLinkQuery, userId, newId, 1)
	if err != nil {
		_ = tx.Rollback(gc)
		gc.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("insert user_organizer_link failed: %v", err)})
		return
	}

	// Commit transaction
	if err = tx.Commit(gc); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit transaction"})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"id":      newId,
		"message": "Organizer created successfully",
	})
}

func OrganizerVenuesHandler(gc *gin.Context) {
	pool := app.Singleton.MainDbPool
	ctx := gc.Request.Context()

	userId, err := app.CurrentUserId(gc)
	if userId < 0 {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get the "id" path parameter
	idStr := gc.Param("id")
	organizerId, err := strconv.Atoi(idStr)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "invalid organizer id"})
		return
	}

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
	fmt.Println("startDate:", startDate)

	// Run query
	row := pool.QueryRow(ctx, app.Singleton.SqlAdminOrganizerVenues, userId, organizerId, startDate)

	var jsonResult []byte
	if err := row.Scan(&jsonResult); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			gc.JSON(http.StatusNoContent, gin.H{"error": err.Error()})
			return
		}
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// The SQL currently returns an array like: [{...}], so we unmarshal and return first element
	var organizers []map[string]interface{}
	if err := json.Unmarshal(jsonResult, &organizers); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse JSON"})
		return
	}

	if len(organizers) == 0 {
		gc.JSON(http.StatusNoContent, gin.H{"error": "organizer not found"})
		return
	}

	// Return only the first organizer as an object
	singleOrganizerJSON, err := json.Marshal(organizers[0])
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encode JSON"})
		return
	}

	gc.Data(http.StatusOK, "application/json", singleOrganizerJSON)
}
