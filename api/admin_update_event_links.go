package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// TODO: Review code

type updateEventLinksRequest struct {
	Links []struct {
		Url     string `json:"url"`
		Title   string `json:"title"`
		UrlType string `json:"url_type"`
	} `json:"links"`
}

func (h *ApiHandler) AdminUpdateEventLinks(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := h.DbPool
	dbSchema := h.Config.DbSchema

	// Get event ID from URL
	eventId := gc.Param("eventId")
	if eventId == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "event ID is required"})
		return
	}

	var req updateEventLinksRequest
	if err := gc.ShouldBindJSON(&req); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	sqlDelete := strings.Replace(`DELETE FROM {{schema}}.event_url WHERE event_id = $1`, "{{schema}}", dbSchema, 1)
	if _, err = tx.Exec(ctx, sqlDelete, eventId); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to delete existing event links: %v", err)})
		return
	}

	sqlInsert := strings.Replace(
		`INSERT INTO {{schema}}.event_url (event_id, url, title, url_type) VALUES ($1, $2, $3, $4)`,
		"{{schema}}", dbSchema, 1,
	)

	for _, link := range req.Links {
		_, err := tx.Exec(ctx, sqlInsert, eventId, link.Url, link.Title, link.UrlType)
		if err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to insertevent link: %v", err)})
			return
		}
	}

	if err = tx.Commit(ctx); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to commit transaction: %v", err)})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"event_id": eventId,
		"message":  "event links updated successfully",
	})
}
