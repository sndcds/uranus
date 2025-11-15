package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) AdminCreateOrganizer(gc *gin.Context) {
	pool := h.DbPool
	ctx := gc.Request.Context()
	userId := gc.GetInt("user-id")

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

	// Begin transaction
	tx, err := pool.Begin(gc)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Insert organizer
	var newId int
	insertOrganizerQuery := `
		INSERT INTO {{schema}}.organizer
			(name, street, house_number, postal_code, city, contact_email, website_url, contact_phone, wkb_geometry)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, ST_GeomFromText($9, 4326))
		RETURNING id
	`
	insertOrganizerQuery = strings.Replace(insertOrganizerQuery, "{{schema}}", h.Config.DbSchema, 1)

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
		gc.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("insert organizer failed: %v", err)})
		return
	}

	// Insert user_organizer_link
	insertLinkQuery := `
		INSERT INTO {{schema}}.user_organizer_link (user_id, organizer_id, permissions)
		VALUES ($1, $2, $3)
	`
	insertLinkQuery = strings.Replace(insertLinkQuery, "{{schema}}", h.Config.DbSchema, 1)

	_, err = tx.Exec(gc, insertLinkQuery, userId, newId, PermissionAdmin)
	if err != nil {
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
