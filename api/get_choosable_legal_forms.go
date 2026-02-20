package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) GetChoosableLegalForms(gc *gin.Context) {
	ctx := gc.Request.Context()
	apiRequest := grains_api.NewRequest(gc, "get_choosable_legal_forms")

	lang := gc.DefaultQuery("lang", "en")
	apiRequest.SetMeta("language", lang)

	query := fmt.Sprintf(
		`SELECT key, name, description FROM %s.legal_form_i18n WHERE iso_639_1 = $1 ORDER BY LOWER(name)`,
		app.UranusInstance.Config.DbSchema,
	)
	rows, err := h.DbPool.Query(ctx, query, lang)
	if err != nil {
		apiRequest.InternalServerError()
		return
	}
	defer rows.Close()

	type LegalForm struct {
		Key         string  `json:"key"`
		Name        *string `json:"name"`
		Description *string `json:"description"`
	}

	var legalForms []LegalForm

	for rows.Next() {
		var legalForm LegalForm
		if err := rows.Scan(
			&legalForm.Key,
			&legalForm.Name,
			&legalForm.Description,
		); err != nil {
			apiRequest.InternalServerError()
			return
		}
		legalForms = append(legalForms, legalForm)
	}

	if err := rows.Err(); err != nil {
		apiRequest.DatabaseError()
		return
	}

	if len(legalForms) == 0 {
		apiRequest.NotFound("No legal forms found")
		return
	}

	apiRequest.SetMeta("legal_form_count", len(legalForms))
	apiRequest.Success(http.StatusOK, legalForms, "")
}
