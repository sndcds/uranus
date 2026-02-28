package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
)

type TransportStationResult struct {
	Id                  int      `json:"id"`
	Name                *string  `json:"name,omitempty"`
	Lon                 *float64 `json:"lon,omitempty"`
	Lat                 *float64 `json:"lat,omitempty"`
	GtfsStationCode     *string  `json:"gtfs_station_code,omitempty"`
	City                *string  `json:"city,omitempty"`
	Country             *string  `json:"country,omitempty"`
	GtfsParentStation   *string  `json:"gtfs_parent_station,omitempty"`
	GtfsWheelchairBoard *int     `json:"gtfs_wheelchair_boarding,omitempty"`
	GtfsZoneId          *string  `json:"gtfs_zone_id,omitempty"`
	DistanceMeters      float64  `json:"distance_m"` // calculated
}

func (h *ApiHandler) GetTransportStations(gc *gin.Context) {
	ctx := gc.Request.Context()
	apiRequest := grains_api.NewRequest(gc, "transport-stations")

	// Parse query params
	latStr := gc.Query("lat")
	lonStr := gc.Query("lon")
	radius := GetContextParamIntDefault(gc, "radius", 5000)
	apiRequest.SetMeta("radius", radius)

	if latStr == "" || lonStr == "" {
		apiRequest.Error(http.StatusBadRequest, "lat and lon are required")
		return
	}
	apiRequest.SetMeta("lat", latStr)
	apiRequest.SetMeta("lon", lonStr)

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		apiRequest.Error(http.StatusBadRequest, "invalid lat")
		return
	}
	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil {
		apiRequest.Error(http.StatusBadRequest, "invalid lon")
		return
	}

	// PostgreSQL query with ST_DWithin and distance
	query := `
	SELECT
		id,
		name,
		gtfs_station_code,
		city,
		country,
		gtfs_parent_station,
		gtfs_wheelchair_boarding,
		gtfs_zone_id,
		ST_X(geo_pos) AS lon,
		ST_Y(geo_pos) AS lat,
		ST_Distance(
			geo_pos::geography,
			ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography
		) AS distance_m
	FROM uranus.transport_station
	WHERE ST_DWithin(
		geo_pos::geography,
		ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography,
		$3
	)
	ORDER BY distance_m;
	`

	rows, err := h.DbPool.Query(ctx, query, lon, lat, radius)
	if err != nil {
		apiRequest.InternalServerError()
		return
	}
	defer rows.Close()

	var results []TransportStationResult
	for rows.Next() {
		var s TransportStationResult
		if err := rows.Scan(
			&s.Id,
			&s.Name,
			&s.GtfsStationCode,
			&s.City,
			&s.Country,
			&s.GtfsParentStation,
			&s.GtfsWheelchairBoard,
			&s.GtfsZoneId,
			&s.Lon,
			&s.Lat,
			&s.DistanceMeters,
		); err != nil {
			apiRequest.InternalServerError()
			return
		}
		results = append(results, s)
	}

	apiRequest.Success(http.StatusOK, results, "")
}
