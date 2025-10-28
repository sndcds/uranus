package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

func ChoosableLegalFormsHandler(gc *gin.Context) {
	db := app.Singleton.MainDbPool
	ctx := gc.Request.Context()

	langStr := gc.DefaultQuery("lang", "en")

	sql := fmt.Sprintf(
		`SELECT legal_form_id, name FROM %s.legal_form WHERE iso_639_1 = $1`,
		app.Singleton.Config.DbSchema,
	)

	rows, err := db.Query(ctx, sql, langStr)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type LegalForm struct {
		LegalFormId int64   `json:"legal_form_id"`
		Name        *string `json:"name"`
	}

	var legalForms []LegalForm

	for rows.Next() {
		var legalForm LegalForm
		if err := rows.Scan(
			&legalForm.LegalFormId,
			&legalForm.Name,
		); err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		legalForms = append(legalForms, legalForm)
	}

	if err := rows.Err(); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(legalForms) == 0 {
		gc.JSON(http.StatusOK, []LegalForm{})
		return
	}

	gc.JSON(http.StatusOK, legalForms)
}
