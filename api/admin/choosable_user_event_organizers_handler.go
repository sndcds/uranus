package api_admin

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

func ChoosableUserEventOrganizersHandler(gc *gin.Context) {
	db := app.Singleton.MainDbPool
	ctx := gc.Request.Context()

	userId, ok := app.GetCurrentUserOrAbort(gc)
	if !ok {
		return // already sent error response
	}

	// Parse organizer ID from path param
	idStr := gc.Param("id")
	organizerId, err := strconv.Atoi(idStr)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sql := app.Singleton.SqlAdminChoosableUserEventOrganizers
	rows, err := db.Query(ctx, sql, userId, organizerId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type Organizer struct {
		Id          int64   `json:"id"`
		Name        *string `json:"name"`
		City        *string `json:"city"`
		CountryCode *string `json:"country_code"`
	}

	var organizers []Organizer

	fmt.Println("organizerId:", organizerId)
	for rows.Next() {
		var ueo Organizer
		if err := rows.Scan(
			&ueo.Id,
			&ueo.Name,
			&ueo.City,
			&ueo.CountryCode,
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

	fmt.Println("organizerId:", organizerId)
	fmt.Println("organizers:", organizers)

	gc.JSON(http.StatusOK, organizers)
}
