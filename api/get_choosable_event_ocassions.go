package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
)

func (h *ApiHandler) GetChoosableEventOccasions(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "get-choosable-event-occasions")
	ctx := gc.Request.Context()

	lang := gc.DefaultQuery("lang", "en")
	apiRequest.SetMeta("language", lang)

	query := fmt.Sprintf(
		`SELECT type_id AS id, name
		 FROM %s.event_occasion_type
		 WHERE iso_639_1 = $1
		 ORDER BY CASE WHEN type_id = 0 THEN 0 ELSE 1 END, name`,
		h.DbSchema,
	)

	rows, err := h.DbPool.Query(ctx, query, lang)
	if err != nil {
		apiRequest.InternalServerError()
		return
	}
	defer rows.Close()

	type EventOccasionType struct {
		Id   *string `json:"id"`
		Name *string `json:"name"`
	}

	var eventOccasions []EventOccasionType

	for rows.Next() {
		var eventOccasion EventOccasionType

		err := rows.Scan(
			&eventOccasion.Id,
			&eventOccasion.Name,
		)
		if err != nil {
			debugf(err.Error())
			apiRequest.InternalServerError()
			return
		}

		eventOccasions = append(eventOccasions, eventOccasion)
	}

	err = rows.Err()
	if err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}

	apiRequest.SetMeta("event_occasion_count", len(eventOccasions))

	if len(eventOccasions) == 0 {
		apiRequest.Success(http.StatusOK, []EventOccasionType{})
		return
	}

	apiRequest.SetMeta("event_occasion_count", len(eventOccasions))
	apiRequest.Success(http.StatusOK, eventOccasions)
}
