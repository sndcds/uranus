package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
)

// TODO: Review code

func (h *ApiHandler) GetChoosableLanguages(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "get-choosable-languages")
	ctx := gc.Request.Context()

	lang := gc.DefaultQuery("lang", "en")
	apiRequest.SetMeta("language", lang)

	query := fmt.Sprintf(
		`SELECT code_iso_639_1, name FROM %s.language WHERE name_iso_639_1 = $1 ORDER BY name`,
		app.UranusInstance.Config.DbSchema,
	)

	rows, err := h.DbPool.Query(ctx, query, lang)
	if err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}
	defer rows.Close()

	type Language struct {
		Id   *string `json:"id"`
		Name *string `json:"name"`
	}

	var languages []Language

	for rows.Next() {
		var language Language
		err := rows.Scan(
			&language.Id,
			&language.Name,
		)
		if err != nil {
			debugf(err.Error())
			apiRequest.InternalServerError()
			return
		}
		languages = append(languages, language)
	}

	err = rows.Err()
	if err != nil {
		debugf(err.Error())
		apiRequest.InvalidJSONInput()
		return
	}

	if len(languages) == 0 {
		apiRequest.Success(http.StatusOK, []Language{}, "")
		return
	}

	apiRequest.Success(http.StatusOK, languages, "")
}
