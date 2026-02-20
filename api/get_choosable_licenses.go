package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
)

func (h *ApiHandler) GetChoosableLicenseTypes(gc *gin.Context) {
	ctx := gc.Request.Context()
	apiRequest := grains_api.NewRequest(gc, "get-chooseable-licenses")

	lang := gc.DefaultQuery("lang", "en")
	apiRequest.SetMeta("language", lang)

	query := fmt.Sprintf(
		`SELECT key, name, description FROM %s.license_i18n WHERE iso_639_1 = $1 ORDER BY key`,
		h.DbSchema)

	rows, err := h.DbPool.Query(ctx, query, lang)
	if err != nil {
		apiRequest.InternalServerError()
		return
	}
	defer rows.Close()

	type License struct {
		Key         *string `json:"key"`
		Name        *string `json:"name"`
		Description *string `json:"description"`
	}

	var licences []License

	for rows.Next() {
		var license License
		if err := rows.Scan(
			&license.Key,
			&license.Name,
			&license.Description,
		); err != nil {
			apiRequest.InternalServerError()
			return
		}
		licences = append(licences, license)
	}

	if err := rows.Err(); err != nil {
		apiRequest.InternalServerError()
		return
	}

	if len(licences) == 0 {
		apiRequest.NotFound("No licenses found")
		return
	}

	apiRequest.SetMeta("license_count", len(licences))
	apiRequest.Success(http.StatusOK, licences, "")
}
