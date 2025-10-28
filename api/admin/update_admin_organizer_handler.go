package api_admin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

type OrganizerUpdateRequest struct {
	Name               string   `json:"name"`
	Description        *string  `json:"description"`
	LegalFormID        *int     `json:"legal_form_id"`
	HoldingOrganizerID *int     `json:"holding_organizer_id"`
	Nonprofit          *bool    `json:"nonprofit"`
	ContactEmail       *string  `json:"contact_email"`
	ContactPhone       *string  `json:"contact_phone"`
	WebsiteURL         *string  `json:"website_url"`
	Street             *string  `json:"street"`
	HouseNumber        *string  `json:"house_number"`
	PostalCode         *string  `json:"postal_code"`
	City               *string  `json:"city"`
	StateCode          *string  `json:"state_code"`
	CountryCode        *string  `json:"country_code"`
	AddressAddition    *string  `json:"address_addition"`
	Longitude          *float64 `json:"longitude"`
	Latitude           *float64 `json:"latitude"`
}

func UpdateAdminOrganizerHandler(gc *gin.Context) {
	pool := app.Singleton.MainDbPool
	ctx := gc.Request.Context()

	organizerID := gc.Param("organizerId")
	if organizerID == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "Organizer ID is required"})
		return
	}

	var req OrganizerUpdateRequest
	if err := gc.ShouldBindJSON(&req); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := pool.Exec(
		ctx,
		app.Singleton.SqlGetAdminOrganizer,
		organizerID,
		req.Name,
		req.Description,
		req.LegalFormID,
		req.HoldingOrganizerID,
		req.Nonprofit,
		req.ContactEmail,
		req.ContactPhone,
		req.WebsiteURL,
		req.Street,
		req.HouseNumber,
		req.PostalCode,
		req.City,
		req.StateCode,
		req.CountryCode,
		req.AddressAddition,
		req.Longitude,
		req.Latitude,
	)

	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	gc.JSON(http.StatusOK, gin.H{"message": "Organizer updated successfully"})
}
