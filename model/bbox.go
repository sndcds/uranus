package model

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type BBox struct {
	MinLon float64
	MinLat float64
	MaxLon float64
	MaxLat float64
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func ParseBBox(s string) (*BBox, error) {
	parts := strings.Split(s, ",")
	if len(parts) != 4 {
		return nil, errors.New("bbox must have 4 values: minLon,minLat,maxLon,maxLat")
	}

	vals := make([]float64, 4)
	for i := 0; i < 4; i++ {
		v, err := strconv.ParseFloat(strings.TrimSpace(parts[i]), 64)
		if err != nil {
			return nil, fmt.Errorf("invalid bbox value at index %d: %w", i, err)
		}
		vals[i] = v
	}

	minLon := clamp(vals[0], -180, 180)
	minLat := clamp(vals[1], -90, 90)
	maxLon := clamp(vals[2], -180, 180)
	maxLat := clamp(vals[3], -90, 90)

	// Normalize if user sends inverted bbox
	if minLon > maxLon {
		minLon, maxLon = maxLon, minLon
	}
	if minLat > maxLat {
		minLat, maxLat = maxLat, minLat
	}

	return &BBox{
		MinLon: minLon,
		MinLat: minLat,
		MaxLon: maxLon,
		MaxLat: maxLat,
	}, nil
}
