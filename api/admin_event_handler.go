package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
	"net/http"
	"strconv"
)

func AdminEventHandler(gc *gin.Context) {
	eventIDStr := gc.Param("id")
	eventID, err := strconv.Atoi(eventIDStr)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{
			"message": fmt.Sprintf("Uranus server error: 400 (Bad Request) %s .. id is not a number", gc.FullPath()),
		})
		return
	}

	db := app.Singleton.MainDbPool
	ctx := gc.Request.Context()

	sql := app.Singleton.SqlAdminEvent
	rows, err := db.Query(ctx, sql, eventID)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	defer rows.Close()

	fieldDescriptions := rows.FieldDescriptions()
	cols := make([]string, len(fieldDescriptions))
	for i, fd := range fieldDescriptions {
		cols[i] = string(fd.Name)
	}

	if !rows.Next() {
		gc.JSON(http.StatusNotFound, gin.H{"message": "event not found"})
		return
	}

	values, err := rows.Values()
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	rowMap := make(map[string]interface{})
	for i, col := range cols {
		if b, ok := values[i].([]byte); ok {
			rowMap[col] = string(b)
		} else {
			rowMap[col] = values[i]
		}
	}

	gc.JSON(http.StatusOK, rowMap)
}
