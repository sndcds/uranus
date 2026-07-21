package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/sql_utils"
)

type venueResponse struct {
	VenueUuid            string  `json:"uuid"`
	OrgUuid              string  `json:"org_uuid"`
	Type                 *string `json:"type"`
	TypeMarkerStyle      *string `json:"type_marker_style"`
	TypeName             *string `json:"type_name"`
	TypeDescription      *string `json:"type_description"`
	Name                 string  `json:"name"`
	Description          *string `json:"description"`
	Summary              *string `json:"summary"`
	ContactEmail         *string `json:"contact_email"`
	ContactPhone         *string `json:"contact_phone"`
	WebLink              *string `json:"web_link"`
	Street               *string `json:"street"`
	HouseNumber          *string `json:"house_number"`
	PostalCode           *string `json:"postal_code"`
	City                 *string `json:"city"`
	State                *string `json:"state"`
	Country              *string `json:"country"`
	Lat                  *string `json:"lat"`
	Lon                  *string `json:"lon"`
	OpenedAt             *string `json:"opened_at"`
	ClosedAt             *string `json:"closed_at"`
	TicketInfo           *string `json:"ticket_info"`
	TicketLink           *string `json:"ticket_link"`
	OpeningHours         *string `json:"opening_hours"`
	AccessibilityFlags   *int64  `json:"accessibility_flags"`
	AccessibilitySummary *string `json:"accessibility_summary"`
	ContentLanguage      *string `json:"content_language"`
	Slug                 *string `json:"slug"`
	Logos                *string `json:"logos"`
	Images               *string `json:"images"`
}

type venuesResponse struct {
	Venues []venueResponse `json:"venues"`
}

type venueFilters struct {
	Args          []interface{}
	ArgIndex      int
	ConditionsStr string
	LimitClause   string
}

func (h *ApiHandler) GetVenuesSummary(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "get-venues-type-summary")
	ctx := gc.Request.Context()

	debugf("GetVenuesSummary")

	filters, err := h.buildVenueFilters(gc, false)
	if err != nil {
		apiRequest.Error(http.StatusBadRequest, err.Error())
		return
	}

	query := app.UranusInstance.SqlGetVenuesSummary
	query = strings.Replace(query, "{{conditions}}", filters.ConditionsStr, 1)

	debugf(query)
	data, err := json.MarshalIndent(filters, "", "  ")
	if err == nil {
		debugf(string(data))
	}

	var summary json.RawMessage
	err = h.DbPool.QueryRow(ctx, query, filters.Args...).Scan(&summary)
	if err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}

	apiRequest.Success(http.StatusOK, summary)
}

func (h *ApiHandler) GetVenues(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "get-venues")
	ctx := gc.Request.Context()

	filters, err := h.buildVenueFilters(gc, true)
	if err != nil {
		apiRequest.Error(http.StatusBadRequest, err.Error())
		return
	}

	query := app.UranusInstance.SqlGetVenues
	query = strings.Replace(query, "{{conditions}}", filters.ConditionsStr, 1)
	query = strings.Replace(query, "{{limit}}", filters.LimitClause, 1)

	debugf(query)
	data, err := json.MarshalIndent(filters, "", "  ")
	if err == nil {
		debugf(string(data))
	}

	rows, err := h.DbPool.Query(ctx, query, filters.Args...)
	if err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}
	defer rows.Close()

	venues := make([]venueResponse, 0)

	for rows.Next() {

		var v venueResponse

		err := rows.Scan(
			&v.VenueUuid,
			&v.OrgUuid,

			&v.Type,
			&v.TypeMarkerStyle,
			&v.TypeName,
			&v.TypeDescription,

			&v.Name,
			&v.Description,
			&v.Summary,

			&v.ContactEmail,
			&v.ContactPhone,
			&v.WebLink,

			&v.Street,
			&v.HouseNumber,
			&v.PostalCode,
			&v.City,
			&v.State,
			&v.Country,

			&v.Lat,
			&v.Lon,

			&v.OpenedAt,
			&v.ClosedAt,

			&v.TicketInfo,
			&v.TicketLink,
			&v.OpeningHours,

			&v.AccessibilityFlags,
			&v.AccessibilitySummary,

			&v.ContentLanguage,
			&v.Slug,

			&v.Logos,
			&v.Images,
		)

		if err != nil {
			debugf(err.Error())
			apiRequest.InternalServerError()
			return
		}

		venues = append(venues, v)
	}

	if err := rows.Err(); err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}

	apiRequest.Success(http.StatusOK, venuesResponse{
		Venues: venues,
	})
}

func (h *ApiHandler) buildVenueFilters(gc *gin.Context, useLang bool) (venueFilters, error) {
	allowed := map[string]struct{}{
		"lang":          {},
		"venues":        {},
		"organizations": {},
		"types":         {},
		"countries":     {},
		"states":        {},
		"cities":        {},
		"postal_code":   {},
		"name":          {},
		"search":        {},
		"accessibility": {},
		"lon":           {},
		"lat":           {},
		"radius":        {},
		"offset":        {},
		"limit":         {},
	}

	filters := venueFilters{}

	if err := validateAllowedQueryParams(gc, allowed); err != nil {
		return filters, err
	}

	filters.Args = []interface{}{}
	filters.ArgIndex = 1

	var conditions []string
	var errBuild error

	if useLang {
		langStr, _ := GetContextParamWithDefault(gc, "lang", "de")
		if !app.ContainsString(h.Config.SupportedLanguages, langStr) {
			return filters, fmt.Errorf("unsupported language: %s", langStr)
		}

		filters.Args = append(filters.Args, langStr)
		filters.ArgIndex++
	}

	venueUuidsStr, _ := GetContextParam(gc, "venue_uuids")
	orgUuidsStr, _ := GetContextParam(gc, "org_uuids")
	typesStr, _ := GetContextParam(gc, "types")
	countriesStr, _ := GetContextParam(gc, "countries")
	citiesStr, _ := GetContextParam(gc, "cities")
	postalCodeStr, _ := GetContextParam(gc, "postal_code")
	nameStr, _ := GetContextParam(gc, "name")
	searchStr, _ := GetContextParam(gc, "search")
	accessibilityStr, _ := GetContextParam(gc, "accessibility")
	lonStr, _ := GetContextParam(gc, "lon")
	latStr, _ := GetContextParam(gc, "lat")
	radiusStr, _ := GetContextParam(gc, "radius")
	offsetStr, _ := GetContextParam(gc, "offset")
	limitStr, _ := GetContextParam(gc, "limit")

	// Uuid filters

	if venueUuidsStr != "" {
		filters.ArgIndex, errBuild = sql_utils.BuildColumnInUuidCondition(
			venueUuidsStr,
			"v.uuid",
			filters.ArgIndex,
			&conditions,
			&filters.Args,
		)
		if errBuild != nil {
			return filters, errBuild
		}
	}

	if orgUuidsStr != "" {
		filters.ArgIndex, errBuild = sql_utils.BuildColumnInUuidCondition(
			orgUuidsStr,
			"v.org_uuid",
			filters.ArgIndex,
			&conditions,
			&filters.Args,
		)
		if errBuild != nil {
			return filters, errBuild
		}
	}

	// Text filters

	if nameStr != "" {
		filters.ArgIndex, errBuild = sql_utils.BuildSanitizedIlikeCondition(
			nameStr,
			"v.name",
			"name",
			filters.ArgIndex,
			&conditions,
			&filters.Args,
		)
		if errBuild != nil {
			return filters, errBuild
		}
	}

	if searchStr != "" {
		filters.ArgIndex, errBuild = sql_utils.BuildSanitizedSearchCondition(
			searchStr,
			"v.search_text",
			"search",
			filters.ArgIndex,
			&conditions,
			&filters.Args,
		)
		if errBuild != nil {
			return filters, errBuild
		}
	}

	if citiesStr != "" {
		filters.ArgIndex, errBuild = sql_utils.BuildSanitizedIlikeCondition(
			citiesStr,
			"v.city",
			"cities",
			filters.ArgIndex,
			&conditions,
			&filters.Args,
		)
		if errBuild != nil {
			return filters, errBuild
		}
	}

	if postalCodeStr != "" {
		filters.ArgIndex, errBuild = sql_utils.BuildLikeConditions(
			postalCodeStr,
			"v.postal_code",
			filters.ArgIndex,
			&conditions,
			&filters.Args,
		)
		if errBuild != nil {
			return filters, errBuild
		}
	}

	// IN filters

	if typesStr != "" {
		filters.ArgIndex, errBuild = sql_utils.BuildInConditionForStringSlice(
			typesStr,
			"v.type = ANY($%d::text[])",
			filters.ArgIndex,
			&conditions,
			&filters.Args,
		)
		if errBuild != nil {
			return filters, errBuild
		}
	}

	if countriesStr != "" {
		filters.ArgIndex, errBuild = sql_utils.BuildInConditionForStringSlice(
			countriesStr,
			"v.country = ANY($%d::text[])",
			filters.ArgIndex,
			&conditions,
			&filters.Args,
		)
		if errBuild != nil {
			return filters, errBuild
		}
	}

	// Accessibility

	if accessibilityStr != "" {
		filters.ArgIndex, errBuild = sql_utils.BuildBitmaskCondition(
			accessibilityStr,
			"v.accessibility_flags",
			"accessibility",
			filters.ArgIndex,
			&conditions,
			&filters.Args,
		)
		if errBuild != nil {
			return filters, errBuild
		}
	}

	// Radius

	filters.ArgIndex, errBuild = sql_utils.BuildGeoRadiusCondition(
		lonStr,
		latStr,
		radiusStr,
		"v.point",
		filters.ArgIndex,
		&conditions,
		&filters.Args,
	)
	if errBuild != nil {
		return filters, errBuild
	}

	// WHERE

	if len(conditions) > 0 {
		filters.ConditionsStr = " AND " + strings.Join(conditions, " AND ")
	}

	// LIMIT/OFFSET

	filters.LimitClause,
		filters.ArgIndex,
		errBuild = sql_utils.BuildLimitOffsetClause(
		limitStr,
		offsetStr,
		filters.ArgIndex,
		&filters.Args,
	)

	if errBuild != nil {
		return filters, errBuild
	}

	return filters, nil
}
