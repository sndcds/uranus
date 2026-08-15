package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

// TODO: Add query parameters for filtering

func (h *ApiHandler) GetVenuesGeoJSON(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "get-venues-geojson")
	ctx := gc.Request.Context()

	apiRequest.SetMeta("api_url", h.Config.BaseApiUrl)

	lang := gc.DefaultQuery("lang", "en")
	apiRequest.SetMeta("language", lang)

	portalUuid := gc.Query("portal")
	if portalUuid != "" {
		apiRequest.SetMeta("portal", portalUuid)
	}

	bboxStr := gc.Query("bbox")
	bbox, err := model.ParseBBox(bboxStr)
	if err != nil {
		apiRequest.Error(http.StatusBadRequest, "invalid bbox")
		return
	}
	apiRequest.SetMeta("bbox", bboxStr)

	// Venue Scopes

	scopesStr := gc.Query("scopes")

	allowedScopes := map[string]bool{
		"shared":       true,
		"organization": true,
	}

	scopes := make([]string, 0)
	if scopesStr != "" {
		for _, value := range strings.Split(scopesStr, ",") {
			scope := strings.TrimSpace(value)

			if scope == "" || !allowedScopes[scope] {
				apiRequest.Error(http.StatusBadRequest, "Invalid scopes")
				return
			}

			scopes = append(scopes, scope)
		}
	}

	apiRequest.SetMeta("scopes", scopes)

	// Query

	var query string
	var rows pgx.Rows

	if portalUuid != "" {
		query = app.UranusInstance.SqlGetPortalVenuesGeoJSON

		rows, err = h.DbPool.Query(
			ctx,
			query,
			bbox.MinLon,
			bbox.MinLat,
			bbox.MaxLon,
			bbox.MaxLat,
			portalUuid,
			scopes,
		)
	} else {
		query = app.UranusInstance.SqlGetVenuesGeoJSON

		rows, err = h.DbPool.Query(
			ctx,
			query,
			bbox.MinLon,
			bbox.MinLat,
			bbox.MaxLon,
			bbox.MaxLat,
			scopes,
		)
	}

	// debugf(query)

	if err != nil {
		debugf(err.Error())
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
			case "lon":
				if v, ok := val.(float64); ok {
					lon = v
				}
			case "lat":
				if v, ok := val.(float64); ok {
					lat = v
				}
			case "count":
				switch v := val.(type) {
				case int:
					props["count"] = v
				case int64:
					props["count"] = int(v)
				case float64:
					props["count"] = int(v)
				default:
					props["count"] = 0
				}
			default:
				props[col] = val
			}
		}

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
		debugf(rows.Err().Error())
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
	apiRequest.Success(http.StatusOK, geojson)
}
