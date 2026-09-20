package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
)

func (h *ApiHandler) GetChoosablePriceTypes(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "get-choosable-price-types")
	ctx := gc.Request.Context()

	lang := gc.DefaultQuery("lang", "en")
	apiRequest.SetMeta("language", lang)

	query := fmt.Sprintf(
		`SELECT type_id AS id, name
		 FROM %s.price_type
		 WHERE iso_639_1 = $1
		 ORDER BY CASE WHEN type_id = 0 THEN 0 ELSE 1 END, name`,
		h.DbSchema,
	)

	rows, err := h.DbPool.Query(ctx, query, lang)
	if err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}
	defer rows.Close()

	type PriceType struct {
		Id   *string `json:"id"`
		Name *string `json:"name"`
	}

	var priceTypes []PriceType

	for rows.Next() {
		var priceType PriceType

		err := rows.Scan(
			&priceType.Id,
			&priceType.Name,
		)
		if err != nil {
			debugf(err.Error())
			apiRequest.InternalServerError()
			return
		}

		priceTypes = append(priceTypes, priceType)
	}

	err = rows.Err()
	if err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}

	apiRequest.SetMeta("price_type_count", len(priceTypes))

	if len(priceTypes) == 0 {
		apiRequest.Success(http.StatusOK, []PriceType{})
		return
	}

	apiRequest.Success(http.StatusOK, priceTypes)
}
