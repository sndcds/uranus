package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) AdminGetChoosableOrganizers(gc *gin.Context) {
	db := app.Singleton.MainDbPool
	ctx := gc.Request.Context()

	userId, ok := app.GetCurrentUserOrAbort(gc)
	if !ok {
		return // already sent error response
	}

	sql := app.Singleton.SqlAdminChoosableOrganizers
	rows, err := db.Query(ctx, sql, userId)
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
		var ueo Organizer
		if err := rows.Scan(
			&ueo.Id,
			&ueo.Name,
		); err != nil {
			fmt.Println(err.Error())
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		organizers = append(organizers, ueo)
	}

	if err := rows.Err(); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(organizers) == 0 {
		// It's better to return an empty array instead of 204 so clients can safely parse it.
		gc.JSON(http.StatusOK, []Organizer{})
		return
	}

	gc.JSON(http.StatusOK, organizers)
}
