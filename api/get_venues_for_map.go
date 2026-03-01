package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
)

// TODO: Review code
// TODO: Add url parameter

func (h *ApiHandler) GetVenuesGeoJSON(gc *gin.Context) {
	ctx := gc.Request.Context()
	apiRequest := grains_api.NewRequest(gc, "get-venues-geojson")

	lang := gc.DefaultQuery("lang", "en")
	apiRequest.SetMeta("language", lang)

	query := app.UranusInstance.SqlGetGeojsonVenues

	// TODO: languageStr, default "en"
	rows, err := h.DbPool.Query(ctx, query, "en")
	if err != nil {
		apiRequest.InternalServerError()
		return
	}
	defer rows.Close()

	// Get column names
	fieldDescriptions := rows.FieldDescriptions()
	columnNames := make([]string, len(fieldDescriptions))
	for i, fd := range fieldDescriptions {
		columnNames[i] = string(fd.Name)
	}

	// Iterate over rows and build JSON
	var venues []map[string]interface{}
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			apiRequest.InternalServerError()
			return
		}

		rowMap := make(map[string]interface{}, len(values))
		for i, col := range columnNames {
			rowMap[col] = values[i]
		}
		venues = append(venues, rowMap)
	}

	if rows.Err() != nil {
		apiRequest.InternalServerError()
		return
	}

	if len(venues) == 0 {
		apiRequest.NoContent("no venues found")
		return
	}
	apiRequest.SetMeta("venues_count", len(venues))

	apiRequest.Success(http.StatusOK, gin.H{"venues": venues}, "")
}
