package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
)

func (h *ApiHandler) GetChoosableLinkTypes(gc *gin.Context) {
	ctx := gc.Request.Context()
	apiRequest := grains_api.NewRequest(gc, "get-choosable-link-types")

	lang := gc.DefaultQuery("lang", "en")
	apiRequest.SetMeta("language", lang)

	query := fmt.Sprintf(
		`SELECT key, name FROM %s.link_type_i18n WHERE iso_639_1 = $1 ORDER BY LOWER(name)`,
		h.DbSchema,
	)

	rows, err := h.DbPool.Query(ctx, query, lang)
	if err != nil {
		apiRequest.InternalServerError()
		return
	}
	defer rows.Close()

	type LinkType struct {
		Key  *string `json:"key"`
		Name *string `json:"name"`
	}

	var linkTypes []LinkType

	for rows.Next() {
		var linkType LinkType
		if err := rows.Scan(
			&linkType.Key,
			&linkType.Name,
		); err != nil {
			fmt.Println(err.Error())
			apiRequest.InternalServerError()
			return
		}
		linkTypes = append(linkTypes, linkType)
	}

	if err := rows.Err(); err != nil {
		apiRequest.InternalServerError()
		return
	}

	if len(linkTypes) == 0 {
		apiRequest.NotFound("No link types found")
		return
	}
	apiRequest.SetMeta("link_type_count", len(linkTypes))

	apiRequest.Success(http.StatusOK, linkTypes, "link types found")
}
