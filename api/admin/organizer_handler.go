package api_admin

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

func AdminOrganizerCreateHandler(gc *gin.Context) {
	pool := app.Singleton.MainDbPool

	fmt.Printf("AdminOrganizerCreateHandler ...")

	type UpdateRequest struct {
		Name         *string `json:"name"`
		Street       *string `json:"street"`
		HouseNumber  *string `json:"house_number"`
		PostalCode   *string `json:"postal_code"`
		City         *string `json:"city"`
		ContactEmail *string `json:"contact_email"`
		WebsiteUrl   *string `json:"website_url"`
		ContactPhone *string `json:"contact_phone"`
	}

	var req UpdateRequest
	if err := gc.ShouldBindJSON(&req); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate

	// Build sql query
	var newID int
	query := `
		INSERT INTO {{schema}}.organizer
			(name, street, house_number, postal_code, city, contact_email, website_url, contact_phone)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`
	query = strings.Replace(query, "{{schema}}", app.Singleton.Config.DbSchema, 1)

	err := pool.QueryRow(gc, query,
		req.Name,
		req.Street,
		req.HouseNumber,
		req.PostalCode,
		req.City,
		req.ContactEmail,
		req.WebsiteUrl,
		req.ContactPhone,
	).Scan(&newID)

	if err != nil {
		gc.JSON(http.StatusOK, gin.H{"message": "Error"})
		return
	}

	gc.JSON(http.StatusOK, gin.H{"message": query})
}
