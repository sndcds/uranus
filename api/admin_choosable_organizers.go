package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) AdminGetChoosableOrganizers(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := gc.GetInt("user-id")

	query := app.Singleton.SqlAdminChoosableOrganizers
	rows, err := h.DbPool.Query(ctx, query, userId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type Organizer struct {
		Id   int64   `json:"organizer_id"`
		Name *string `json:"organizer_name"`
	}

	var organizers []Organizer

	for rows.Next() {
		var organizer Organizer
		err := rows.Scan(&organizer.Id, &organizer.Name)
		if err != nil {
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
		gc.JSON(http.StatusOK, []Organizer{}) // Returns empty array
		return
	}

	gc.JSON(http.StatusOK, organizers)
}
