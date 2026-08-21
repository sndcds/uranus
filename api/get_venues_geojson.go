package api

import (
	"encoding/json"
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

	allowedScopes := map[string]bool{
		"shared":       true,
		"organization": true,
	}

	scopesStr := gc.Query("scopes")

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
		Point      map[string]interface{} `json:"point"`
		Building   map[string]interface{} `json:"building,omitempty"`
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
		var point map[string]interface{}
		var building map[string]interface{}

		for i, col := range columnNames {
			val := values[i]

			switch col {
			case "point":
				if val != nil {
					pointJSON, ok := val.(string)
					if !ok {
						apiRequest.InternalServerError()
						return
					}

					if err := json.Unmarshal([]byte(pointJSON), &point); err != nil {
						apiRequest.InternalServerError()
						return
					}
				}

			case "building":
				if val != nil {
					buildingJSON, ok := val.(string)
					if !ok {
						apiRequest.InternalServerError()
						return
					}

					if err := json.Unmarshal([]byte(buildingJSON), &building); err != nil {
						apiRequest.InternalServerError()
						return
					}
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
			Type:       "Feature",
			Point:      point,
			Building:   building,
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
