package app

import "fmt"

// GenerateWKT takes lat/lon strings and returns a WKT POINT string
func GenerateWKT(lat, lon float64) (string, error) {
	wkt := fmt.Sprintf("POINT(%f %f)", lon, lat)
	return wkt, nil
}
