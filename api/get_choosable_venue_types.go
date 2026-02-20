package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
)

func (h *ApiHandler) GetChoosableVenueTypes(gc *gin.Context) {
	ctx := gc.Request.Context()
	apiRequest := grains_api.NewRequest(gc, "get-choosable-venue-types")

	// Get language from query parameter, default to "en"
	lang := gc.DefaultQuery("lang", "en")
	apiRequest.SetMeta("language", lang)

	// Query all venue types for the given language
	query := fmt.Sprintf(
		`SELECT vt.key, vti.name, vti.description
         FROM %s.venue_type_i18n vti
         JOIN %s.venue_type vt ON vt.key = vti.key
         WHERE vti.iso_639_1 = $1
         ORDER BY vti.name`,
		h.DbSchema,
		h.DbSchema,
	)

	rows, err := h.DbPool.Query(ctx, query, lang)
	if err != nil {
		apiRequest.DatabaseError()
		return
	}
	defer rows.Close()

	type VenueType struct {
		Key         string  `json:"key"`
		Name        *string `json:"name"`
		Description *string `json:"description"`
	}

	var venueTypes []VenueType

	for rows.Next() {
		var vt VenueType
		if err := rows.Scan(&vt.Key, &vt.Name, &vt.Description); err != nil {
			apiRequest.DatabaseError()
			return
		}
		venueTypes = append(venueTypes, vt)
	}

	if err := rows.Err(); err != nil {
		apiRequest.DatabaseError()
		return
	}

	if len(venueTypes) == 0 {
		apiRequest.NotFound("No venue types found")
		return
	}

	apiRequest.SetMeta("venue_type_count", len(venueTypes))
	apiRequest.Success(http.StatusOK, venueTypes, "")
}
