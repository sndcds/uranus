package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func (h *ApiHandler) AdminGetMessages(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := h.DbPool

	userId := UserIdFromAccessToken(gc)
	if userId == 0 {
		gc.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	sql := fmt.Sprintf(`
		SELECT id, from_user_id, created_at, is_read, subject, message
		FROM %s.message
		WHERE to_user_id = $1
		ORDER BY created_at DESC
	`, h.Config.DbSchema)

	rows, err := pool.Query(ctx, sql, userId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("query failed: %v", err)})
		return
	}
	defer rows.Close()

	type Message struct {
		Id         int       `json:"id"`
		FromUserId int       `json:"from_user_id"`
		CreatedAt  time.Time `json:"created_at"`
		IsRead     bool      `json:"is_read"`
		Subject    string    `json:"subject"`
		Message    string    `json:"message"`
	}

	var messages []Message

	for rows.Next() {
		var msg Message
		if err := rows.Scan(
			&msg.Id,
			&msg.FromUserId,
			&msg.CreatedAt,
			&msg.IsRead,
			&msg.Subject,
			&msg.Message,
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
