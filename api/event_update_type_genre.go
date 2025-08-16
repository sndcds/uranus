package api

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"log"
	"net/http"
)

/*
func updateEventTypeAndGenreHandler(gc *gin.Context) {
	httpCode, err := updateEventTypeAndGenre(gc, app.Singleton.MainDbPool)
	if err != nil {
		gc.JSON(httpCode, gin.H{"error": err.Error()})
		return
	}
	gc.JSON(http.StatusOK, gin.H{"success": "event type and genre are updated in db)"})
}
*/

func updateEventTypeAndGenre(
	gc *gin.Context,
	pool *pgxpool.Pool) (int, error) {

	// Prepare data from JSON
	type UpdateEventLinksRequest struct {
		EventID      int   `json:"event_id"`
		EventTypeIDs []int `json:"event_type_ids"`
		GenreTypeIDs []int `json:"event_genre_ids"`
	}
	var req UpdateEventLinksRequest
	if err := gc.ShouldBindJSON(&req); err != nil {
		return http.StatusBadRequest, fmt.Errorf("invalid input")
	}

	/* Debug code */
	jsonBytes, _ := json.MarshalIndent(req, "", "  ")
	log.Println("Received JSON:\n", string(jsonBytes))
	//*/

	// Transaction
	tx, err := pool.Begin(gc)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback(gc)

	canEditEvent, err := UserCanEditEvent(gc, tx, req.EventID)
	if (!canEditEvent) || (err != nil) {
		return http.StatusInternalServerError, err
	}

	// Validate event_type_ids exist
	ok, err := allIDsExist(gc, tx, "event_type", req.EventTypeIDs)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("some event_type_ids do not exist")
	}

	// Validate event_genre_ids exist
	ok, err = allIDsExist(gc, tx, "event_genre", req.GenreTypeIDs)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("some event_genre_ids do not exist")
	}

	// Delete existing links
	_, err = tx.Exec(gc, "DELETE FROM uranus.event_type_links WHERE event_id = $1", req.EventID)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	_, err = tx.Exec(gc, "DELETE FROM uranus.event_genre_links WHERE event_id = $1", req.EventID)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	// Insert new event types
	for _, typeID := range req.EventTypeIDs {
		_, err = tx.Exec(gc, "INSERT INTO uranus.event_type_links (event_id, type_id) VALUES ($1, $2)", req.EventID, typeID)
		if err != nil {
			return http.StatusInternalServerError, err
		}
	}

	// Insert new genre types
	for _, genreID := range req.GenreTypeIDs {
		_, err = tx.Exec(gc, "INSERT INTO uranus.event_genre_links (event_id, type_id) VALUES ($1, $2)", req.EventID, genreID)
		if err != nil {
			return http.StatusInternalServerError, err
		}
	}

	err = tx.Commit(gc)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}
