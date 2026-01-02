package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

// TODO: Review code

func (h *ApiHandler) GetChoosableLanguages(gc *gin.Context) {
	db := app.UranusInstance.MainDbPool
	ctx := gc.Request.Context()

	lang := gc.DefaultQuery("lang", "en")

	sql := fmt.Sprintf(
		`SELECT code_iso_639_1, name FROM %s.language WHERE name_iso_639_1 = $1 ORDER BY name`,
		app.UranusInstance.Config.DbSchema,
	)

	rows, err := db.Query(ctx, sql, lang)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		if err := rows.Scan(
			&language.Id,
			&language.Name,
		); err != nil {
			fmt.Println(err.Error())
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		languages = append(languages, language)
	}

	if err := rows.Err(); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(languages) == 0 {
		gc.JSON(http.StatusOK, []Language{})
		return
	}

	gc.JSON(http.StatusOK, languages)
}
