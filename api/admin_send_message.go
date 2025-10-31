package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func (h *ApiHandler) AdminSendMessage(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := h.DBPool

	userId := UserIdFromAccessToken(gc)
	if userId == 0 {
		gc.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	type MessageRequest struct {
		Context   string `json:"context" binding:"required"`
		ContextID int    `json:"context_id" binding:"required"`
		Subject   string `json:"subject" binding:"required"`
		Message   string `json:"message" binding:"required"`
	}

	var req MessageRequest
	if err := gc.ShouldBindJSON(&req); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sql := fmt.Sprintf(`
		INSERT INTO %s.message (context, context_id, subject, message, from_user_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at;
	`, h.Config.DbSchema)

	var id int
	var createdAt time.Time

	err := pool.QueryRow(ctx, sql,
		req.Context,
		req.ContextID,
		req.Subject,
		req.Message,
		userId,
	).Scan(&id, &createdAt)

	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"id":         id,
		"created_at": createdAt,
		"context":    req.Context,
		"context_id": req.ContextID,
		"subject":    req.Subject,
		"message":    req.Message,
		"created_by": userId,
	})
}
