package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

func ChoosableLanguagesHandler(gc *gin.Context) {
	db := app.Singleton.MainDbPool
	ctx := gc.Request.Context()

	lang := gc.DefaultQuery("lang", "en")

	sql := fmt.Sprintf(
		`SELECT code_iso_639_1, name FROM %s.language WHERE name_iso_639_1 = $1 ORDER BY name`,
		app.Singleton.Config.DbSchema,
	)

	rows, err := db.Query(ctx, sql, lang)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type Language struct {
		LanguageCode *string `json:"language_code"`
		LanguageName *string `json:"language_name"`
	}

	var languages []Language

	for rows.Next() {
		var language Language
		if err := rows.Scan(
			&language.LanguageCode,
			&language.LanguageName,
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
