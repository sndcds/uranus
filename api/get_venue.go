package api

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
)

// GetVenue returns a venue by Id with spaces and organization
func (h *ApiHandler) GetVenue(gc *gin.Context) {
	ctx := gc.Request.Context()
	apiRequest := grains_api.NewRequest(gc, "ger-venue")

	// Structs for nested data
	type SpaceResult struct {
		Id              int      `json:"id"`
		Name            *string  `json:"name,omitempty"`
		TotalCapacity   *int     `json:"total_capacity,omitempty"`
		SeatingCapacity *int     `json:"seating_capacity,omitempty"`
		BuildingLevel   *int     `json:"building_level,omitempty"`
		WebsiteLink     *string  `json:"website_link,omitempty"`
		Description     *string  `json:"description,omitempty"`
		AreaSqm         *float64 `json:"area_sqm,omitempty"`
		SpaceType       *string  `json:"space_type,omitempty"`
		SpaceTypeName   *string  `json:"space_type_name,omitempty"`
		SpaceTypeDesc   *string  `json:"space_type_description,omitempty"`
	}

	type OrganizationResult struct {
		Id          int     `json:"id"`
		Name        *string `json:"name,omitempty"`
		WebsiteLink *string `json:"website_link,omitempty"`
		City        *string `json:"city,omitempty"`
		Country     *string `json:"country,omitempty"`
	}

	type VenueResult struct {
		Id                   int                 `json:"id"`
		Name                 *string             `json:"name,omitempty"`
		Type                 *string             `json:"type,omitempty"`
		TypeName             *string             `json:"type_name,omitempty"`
		TypeDescription      *string             `json:"type_description,omitempty"`
		OpenedAt             *string             `json:"opened_at,omitempty"`
		ClosedAt             *string             `json:"closed_at,omitempty"`
		Summary              *string             `json:"summary,omitempty"`
		Description          *string             `json:"description,omitempty"`
		Street               *string             `json:"street,omitempty"`
		HouseNumber          *string             `json:"house_number,omitempty"`
		PostalCode           *string             `json:"postal_code,omitempty"`
		City                 *string             `json:"city,omitempty"`
		Country              *string             `json:"country,omitempty"`
		State                *string             `json:"state,omitempty"`
		ContactEmail         *string             `json:"contact_email,omitempty"`
		ContactPhone         *string             `json:"contact_phone,omitempty"`
		WebsiteLink          *string             `json:"website_link,omitempty"`
		TicketLink           *string             `json:"ticket_link,omitempty"`
		TicketInfo           *string             `json:"ticket_info,omitempty"`
		Lon                  *float64            `json:"lon,omitempty"`
		Lat                  *float64            `json:"lat,omitempty"`
		AccessibilityFlags   *string             `json:"accessibility_flags,omitempty"`
		AccessibilitySummary *string             `json:"accessibility_summary,omitempty"`
		Organization         *OrganizationResult `json:"organization,omitempty"`
		Spaces               []SpaceResult       `json:"spaces,omitempty"`
	}

	venueId, ok := ParamInt(gc, "venueId")
	if !ok {
		apiRequest.Error(http.StatusBadRequest, "venue Id is required")
		return
	}

	lang := gc.DefaultQuery("lang", "en")
	apiRequest.SetMeta("language", lang)

	query := app.UranusInstance.SqlGetVenue

	row := h.DbPool.QueryRow(ctx, query, venueId, lang)

	// Temporary variables for SQL scan
	var venue VenueResult
	var org OrganizationResult
	var spacesJSON []byte

	err := row.Scan(
		&venue.Id,
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
		&venue.WebsiteLink,
		&venue.TicketLink,
		&venue.TicketInfo,
		&venue.Lon,
		&venue.Lat,
		&venue.AccessibilityFlags,
		&venue.AccessibilitySummary,
		&org.Id,
		&org.Name,
		&org.WebsiteLink,
		&org.City,
		&org.Country,
		&spacesJSON,
	)
	if err != nil {
		debugf("GetVenue error: %v", err)
		apiRequest.SetMeta("err_code", "1001")
		apiRequest.InternalServerError()
		return
	}

	// Assign organization if any fields are non-nil
	if org.Name != nil || org.WebsiteLink != nil || org.City != nil || org.Country != nil {
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

	apiRequest.Success(http.StatusOK, venue, "")
}
