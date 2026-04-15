package api

import (
	"math/rand"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
)

// TODO: Add query parameters for filtering

func (h *ApiHandler) GetVenuesGeoJSON(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "get-venues-geojson")
	ctx := gc.Request.Context()

	lang := gc.DefaultQuery("lang", "en")
	apiRequest.SetMeta("language", lang)

	query := app.UranusInstance.SqlGetGeojsonVenues

	rows, err := h.DbPool.Query(ctx, query, lang)
	if err != nil {
		apiRequest.InternalServerError()
		return
	}
	defer rows.Close()

	fieldDescriptions := rows.FieldDescriptions()
	columnNames := make([]string, len(fieldDescriptions))

	for i, fd := range fieldDescriptions {
		columnNames[i] = string(fd.Name)
	}

	type Feature struct {
		Type       string                 `json:"type"`
		Geometry   map[string]interface{} `json:"geometry"`
		Properties map[string]interface{} `json:"properties"`
	}

	var features []Feature

	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			apiRequest.InternalServerError()
			return
		}

		props := make(map[string]interface{})
		var lon, lat float64

		for i, col := range columnNames {
			val := values[i]

			switch col {
			case "longitude", "lon", "lng":
				if v, ok := val.(float64); ok {
					lon = v
				}
			case "latitude", "lat":
				if v, ok := val.(float64); ok {
					lat = v
				}
			default:
				props[col] = val
			}
		}

		min := 1
		max := 99
		props["count"] = rand.Intn(max-min+1) + min

		features = append(features, Feature{
			Type: "Feature",
			Geometry: map[string]interface{}{
				"type":        "Point",
				"coordinates": []float64{lon, lat},
			},
			Properties: props,
		})
	}

	if rows.Err() != nil {
		apiRequest.InternalServerError()
		return
	}

	if len(features) == 0 {
		apiRequest.NoContent("no venues found")
		return
	}

	geojson := map[string]interface{}{
		"type":     "FeatureCollection",
		"features": features,
	}

	apiRequest.SetMeta("venues_count", len(features))
	apiRequest.Success(http.StatusOK, geojson, "")
}
