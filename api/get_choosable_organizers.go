package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) GetChoosableOrganizers(gc *gin.Context) {
	db := app.Singleton.MainDbPool
	ctx := gc.Request.Context()

	sql := fmt.Sprintf("SELECT id, name FROM %s.organizer ORDER BY LOWER(name)", h.Config.DbSchema)
	rows, err := db.Query(ctx, sql)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type Organizer struct {
		Id   int64   `json:"id"`
		Name *string `json:"name"`
	}

	var organizers []Organizer

	for rows.Next() {
		var organizer Organizer
		if err := rows.Scan(
			&organizer.Id,
			&organizer.Name,
		); err != nil {
			fmt.Println(err.Error())
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		organizers = append(organizers, organizer)
	}

	if err := rows.Err(); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(organizers) == 0 {
		gc.JSON(http.StatusOK, []Organizer{})
		return
	}

	gc.JSON(http.StatusOK, organizers)
}
