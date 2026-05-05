package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
)

func (h *ApiHandler) GetVenueSpaceLabel(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "get-venue-space-label")
	ctx := gc.Request.Context()

	venueUuid := gc.Param("venueUuid")
	if venueUuid == "" {
		apiRequest.Required("parameter venueUuid is required")
		return
	}

	spaceUuid := gc.Param("spaceUuid")
	if spaceUuid == "" {
		apiRequest.Required("parameter spaceUuid is required")
		return
	}

	apiRequest.SetMeta("venue_uuid", venueUuid)
	apiRequest.SetMeta("space_uuid", spaceUuid)

	var spaceUuidParam *string
	if spaceUuid != "-" {
		spaceUuidParam = &spaceUuid
	}

	query := fmt.Sprintf(`
		SELECT v.name AS venue_name, s.name AS space_name
		FROM %s.venue v
		LEFT JOIN %s.space s ON s.uuid = $2::uuid
		WHERE v.uuid = $1::uuid LIMIT 1`,
		h.DbSchema, h.DbSchema)

	rows, err := h.DbPool.Query(ctx, query, venueUuid, spaceUuidParam)
	if err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}
	defer rows.Close()

	if !rows.Next() {
		apiRequest.NotFound("venue or space not found")
		return
	}

	fieldDescriptions := rows.FieldDescriptions()
	values, err := rows.Values()
	if err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}

	response := make(map[string]interface{}, len(values))
	for i, fd := range fieldDescriptions {
		if values[i] != nil {
			response[string(fd.Name)] = values[i]
		}
	}

	apiRequest.Success(http.StatusOK, response, "")
}
