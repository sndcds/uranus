package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

// TODO: Review code

func (h *ApiHandler) GetChoosableOrganizations(gc *gin.Context) {
	db := app.UranusInstance.MainDbPool
	ctx := gc.Request.Context()

	sql := fmt.Sprintf("SELECT id, name FROM %s.organization ORDER BY LOWER(name)", h.Config.DbSchema)
	rows, err := db.Query(ctx, sql)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type Organization struct {
		Id   int64   `json:"id"`
		Name *string `json:"name"`
	}

	var organizations []Organization

	for rows.Next() {
		var organization Organization
		if err := rows.Scan(
			&organization.Id,
			&organization.Name,
		); err != nil {
			fmt.Println(err.Error())
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		organizations = append(organizations, organization)
	}

	if err := rows.Err(); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(organizations) == 0 {
		gc.JSON(http.StatusOK, []Organization{})
		return
	}

	gc.JSON(http.StatusOK, organizations)
}
