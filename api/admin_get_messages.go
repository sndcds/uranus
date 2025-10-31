package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func (h *ApiHandler) AdminGetMessages(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := h.DBPool

	userId := UserIdFromAccessToken(gc)
	if userId == 0 {
		gc.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	sql := fmt.Sprintf(`
		SELECT id, context, context_id, subject, message, created_at, modified_at, from_user_id, is_read
		FROM %s.message
		WHERE context_id = $1
		ORDER BY created_at DESC
	`, h.Config.DbSchema)

	rows, err := pool.Query(ctx, sql, userId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("query failed: %v", err)})
		return
	}
	defer rows.Close()

	type Message struct {
		ID         int        `json:"id"`
		Context    string     `json:"context"`
		ContextID  int        `json:"context_id"`
		Subject    string     `json:"subject"`
		Message    string     `json:"message"`
		CreatedAt  time.Time  `json:"created_at"`
		ModifiedAt *time.Time `json:"modified_at,omitempty"`
		FromUserID int        `json:"from_user_id"`
		IsRead     bool       `json:"is_read"`
	}

	var messages []Message

	for rows.Next() {
		var msg Message
		if err := rows.Scan(
			&msg.ID,
			&msg.Context,
			&msg.ContextID,
			&msg.Subject,
			&msg.Message,
			&msg.CreatedAt,
			&msg.ModifiedAt,
			&msg.FromUserID,
			&msg.IsRead,
		); err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("scan failed: %v", err)})
			return
		}
		messages = append(messages, msg)
	}

	if rows.Err() != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("rows error: %v", rows.Err())})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"messages": messages,
	})
}
