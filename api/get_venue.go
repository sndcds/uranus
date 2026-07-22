package api

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

// GetVenue returns a venue by Id with spaces and organization
func (h *ApiHandler) GetVenue(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "get-venue")
	ctx := gc.Request.Context()

	// Structs for nested data
	type SpaceResult struct {
		Uuid            *string  `json:"uuid"`
		Name            *string  `json:"name,omitempty"`
		TotalCapacity   *int     `json:"total_capacity,omitempty"`
		SeatingCapacity *int     `json:"seating_capacity,omitempty"`
		BuildingLevel   *int     `json:"building_level,omitempty"`
		WebLink         *string  `json:"web_link,omitempty"`
		Description     *string  `json:"description,omitempty"`
		AreaSqm         *float64 `json:"area_sqm,omitempty"`
		SpaceType       *string  `json:"space_type,omitempty"`
		SpaceTypeName   *string  `json:"space_type_name,omitempty"`
		SpaceTypeDesc   *string  `json:"space_type_description,omitempty"`
	}

	type OrganizationResult struct {
		Uuid    *string `json:"uuid"`
		Name    *string `json:"name,omitempty"`
		WebLink *string `json:"web_link,omitempty"`
		City    *string `json:"city,omitempty"`
		Country *string `json:"country,omitempty"`
	}

	type VenueResult struct {
		Uuid                 *string                `json:"id"`
		Name                 *string                `json:"name,omitempty"`
		Type                 *string                `json:"type,omitempty"`
		TypeName             *string                `json:"type_name,omitempty"`
		TypeDescription      *string                `json:"type_description,omitempty"`
		OpenedAt             *string                `json:"opened_at,omitempty"`
		ClosedAt             *string                `json:"closed_at,omitempty"`
		Summary              *string                `json:"summary,omitempty"`
		Description          *string                `json:"description,omitempty"`
		Street               *string                `json:"street,omitempty"`
		HouseNumber          *string                `json:"house_number,omitempty"`
		PostalCode           *string                `json:"postal_code,omitempty"`
		City                 *string                `json:"city,omitempty"`
		Country              *string                `json:"country,omitempty"`
		State                *string                `json:"state,omitempty"`
		ContactEmail         *string                `json:"contact_email,omitempty"`
		ContactPhone         *string                `json:"contact_phone,omitempty"`
		WebLink              *string                `json:"web_link,omitempty"`
		TicketLink           *string                `json:"ticket_link,omitempty"`
		TicketInfo           *string                `json:"ticket_info,omitempty"`
		Lon                  *float64               `json:"lon,omitempty"`
		Lat                  *float64               `json:"lat,omitempty"`
		AccessibilityFlags   *string                `json:"accessibility_flags,omitempty"`
		AccessibilitySummary *string                `json:"accessibility_summary,omitempty"`
		Organization         *OrganizationResult    `json:"organization,omitempty"`
		Spaces               []SpaceResult          `json:"spaces,omitempty"`
		Logos                map[string]model.Logo  `json:"logos"`
		Images               map[string]model.Image `json:"images,omitempty"`
	}

	venueUuid := gc.Param("venueUuid")
	if venueUuid == "" {
		apiRequest.Required("venueUuid is required")
		return
	}

	lang := gc.DefaultQuery("lang", "en")
	apiRequest.SetMeta("language", lang)

	query := app.UranusInstance.SqlGetVenue

	row := h.DbPool.QueryRow(ctx, query, venueUuid, lang)

	// Temporary variables for SQL scan

	var (
		venue      VenueResult
		org        OrganizationResult
		spacesJSON []byte
		logosJSON  []byte
		imagesJSON []byte
	)

	err := row.Scan(
		&venue.Uuid,
		&venue.Name,
		&venue.Type,
		&venue.TypeName,
		&venue.TypeDescription,
		&venue.OpenedAt,
		&venue.ClosedAt,
		&venue.Summary,
		&venue.Description,
		&venue.Street,
		&venue.HouseNumber,
		&venue.PostalCode,
		&venue.City,
		&venue.Country,
		&venue.State,
		&venue.ContactEmail,
		&venue.ContactPhone,
		&venue.WebLink,
		&venue.TicketLink,
		&venue.TicketInfo,
		&venue.Lon,
		&venue.Lat,
		&venue.AccessibilityFlags,
		&venue.AccessibilitySummary,
		&org.Uuid,
		&org.Name,
		&org.WebLink,
		&org.City,
		&org.Country,
		&spacesJSON,
		&logosJSON,
		&imagesJSON,
	)
	if err != nil {
		debugf(err.Error())
		apiRequest.SetMeta("err_code", "1001")
		apiRequest.InternalServerError()
		return
	}

	// Assign organization if any fields are non-nil
	if org.Name != nil || org.WebLink != nil || org.City != nil || org.Country != nil {
		venue.Organization = &org
	}

	// Decode spaces JSON into structs
	if len(spacesJSON) > 0 {
		if err := json.Unmarshal(spacesJSON, &venue.Spaces); err != nil {
			apiRequest.SetMeta("err_code", "1002")
			apiRequest.InternalServerError()
			return
		}
	}

	if len(logosJSON) > 0 && string(logosJSON) != "null" {
		_ = json.Unmarshal(logosJSON, &venue.Logos)
	}

	if len(imagesJSON) > 0 && string(imagesJSON) != "null" {
		var images map[string]model.Image
		if err := json.Unmarshal(imagesJSON, &images); err == nil {
			venue.Images = images
		}
	}

	apiRequest.Success(http.StatusOK, venue)
}

func imageURL(uuid *string) *string {
	if uuid == nil {
		return nil
	}

	url := ImageUrl(*uuid)
	return &url
}
