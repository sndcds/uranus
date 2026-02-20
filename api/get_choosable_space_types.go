package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
)

func (h *ApiHandler) GetChoosableSpaceTypes(gc *gin.Context) {
	ctx := gc.Request.Context()
	apiRequest := grains_api.NewRequest(gc, "get-choosable-space-types")

	// Get language from query parameter, default to "en"
	lang := gc.DefaultQuery("lang", "en")
	apiRequest.SetMeta("language", lang)

	// Query all space types for the given language
	query := fmt.Sprintf(
		`SELECT st.key, sti.name, sti.description
         FROM %s.space_type_i18n sti
         JOIN %s.space_type st ON st.key = sti.key
         WHERE sti.iso_639_1 = $1
         ORDER BY sti.name`,
		h.DbSchema,
		h.DbSchema,
	)

	rows, err := h.DbPool.Query(ctx, query, lang)
	if err != nil {
		apiRequest.DatabaseError()
		return
	}
	defer rows.Close()

	type SpaceType struct {
		Key         string  `json:"key"`
		Name        *string `json:"name"`
		Description *string `json:"description"`
	}

	var spaceTypes []SpaceType

	for rows.Next() {
		var st SpaceType
		if err := rows.Scan(&st.Key, &st.Name, &st.Description); err != nil {
			apiRequest.DatabaseError()
			return
		}
		spaceTypes = append(spaceTypes, st)
	}

	if err := rows.Err(); err != nil {
		apiRequest.DatabaseError()
		return
	}

	if len(spaceTypes) == 0 {
		apiRequest.NotFound("No space types found")
		return
	}

	apiRequest.SetMeta("space_type_count", len(spaceTypes))
	apiRequest.Success(http.StatusOK, spaceTypes, "")
}
