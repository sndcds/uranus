package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

func ChoosableLicensesHandler(gc *gin.Context) {
	db := app.Singleton.MainDbPool
	ctx := gc.Request.Context()

	langStr := gc.DefaultQuery("lang", "en")

	sql := fmt.Sprintf(
		`SELECT license_id, name, short_name FROM %s.license_type WHERE iso_639_1 = $1`,
		app.Singleton.Config.DbSchema,
	)

	rows, err := db.Query(ctx, sql, langStr)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type License struct {
		LicenseId int64   `json:"license_id"`
		Name      *string `json:"license_name"`
		ShortName *string `json:"short_name"`
	}

	var licenses []License

	for rows.Next() {
		var license License
		if err := rows.Scan(
			&license.LicenseId,
			&license.Name,
			&license.ShortName,
		); err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		licenses = append(licenses, license)
	}

	if err := rows.Err(); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(licenses) == 0 {
		gc.JSON(http.StatusOK, []License{})
		return
	}

	gc.JSON(http.StatusOK, licenses)
}
