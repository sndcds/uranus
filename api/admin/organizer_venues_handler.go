package api_admin

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/uranus/app"
)

func OrganizerVenuesHandler(gc *gin.Context) {
	pool := app.Singleton.MainDbPool
	ctx := gc.Request.Context()

	userId, err := app.CurrentUserId(gc)
	if userId < 0 {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	idStr := gc.Param("id")
	organizerId, err := strconv.Atoi(idStr)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "invalid organizer id"})
		return
	}

	startStr := gc.Query("start")
	var startDate time.Time
	if startStr != "" {
		startDate, err = time.Parse("2006-01-02", startStr)
		if err != nil {
			startDate = time.Now() // fallback on parse error
		}
	} else {
		startDate = time.Now() // fallback if param missing
	}
	fmt.Println("startDate:", startDate)

	// Run query
	row := pool.QueryRow(ctx, app.Singleton.SqlAdminOrganizerVenues, userId, organizerId, startDate)

	var jsonResult []byte
	if err := row.Scan(&jsonResult); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			gc.JSON(http.StatusNoContent, gin.H{"error": err.Error()})
			return
		}
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// The SQL currently returns an array like: [{...}], so we unmarshal and return first element
	var organizers []map[string]interface{}
	if err := json.Unmarshal(jsonResult, &organizers); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse JSON"})
		return
	}

	if len(organizers) == 0 {
		gc.JSON(http.StatusNoContent, gin.H{"error": "organizer not found"})
		return
	}

	// Return only the first organizer as an object
	singleOrganizerJSON, err := json.Marshal(organizers[0])
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encode JSON"})
		return
	}

	gc.Data(http.StatusOK, "application/json", singleOrganizerJSON)
}
