package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

func ChoosableEventGenresHandler(gc *gin.Context) {
	ctx := gc.Request.Context()
	db := app.Singleton.MainDbPool
	sql := app.Singleton.SqlChoosableEventGenres

	idStr := gc.Param("id")
	eventTypeId, err := strconv.Atoi(idStr)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	langStr := gc.DefaultQuery("lang", "en")
	rows, err := db.Query(ctx, sql, eventTypeId, langStr)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type Genre struct {
		TypeId int    `json:"id"`
		Name   string `json:"name"`
	}

	var genres []Genre

	for rows.Next() {
		var genre Genre
		if err := rows.Scan(&genre.TypeId, &genre.Name); err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		genres = append(genres, genre)
	}

	if err := rows.Err(); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	gc.JSON(http.StatusOK, genres)
}
