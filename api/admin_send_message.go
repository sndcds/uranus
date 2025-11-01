package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func (h *ApiHandler) AdminSendMessage(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := h.DbPool

	fromuserId := UserIdFromAccessToken(gc)
	if fromuserId <= 0 {
		gc.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	type MessageRequest struct {
		Context   string `json:"context" binding:"required"`
		ContextId int    `json:"context_id" binding:"required"`
		Subject   string `json:"subject" binding:"required"`
		Message   string `json:"message" binding:"required"`
	}

	var req MessageRequest
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

	if req.Context == "organizer" {
		organizerId := req.ContextId
		sql := strings.Replace(
			`SELECT u.id, u.display_name
			FROM {{schema}}.user_organizer_links ol
			JOIN {{schema}}.user u ON u.id = ol.user_id
			WHERE ol.organizer_id = $1 AND (user_role_id = 1 OR user_role_id = 2)`,
			"{{schema}}", h.Config.DbSchema, -1)

		fmt.Println(sql)
		fmt.Println("organizerId:", organizerId)
		rows, err := tx.Query(ctx, sql, organizerId)
		if err != nil {
			gc.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("database query failed: %v", err)})
			return
		}
		defer rows.Close()

		var userIds []int
		for rows.Next() {
			var id int
			var displayName *string
			if err := rows.Scan(&id, &displayName); err != nil {
				gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to scan row: %v", err)})
				return
			}
			fmt.Println(id, displayName)
			userIds = append(userIds, id)

			// TODO: Duplikate verhindern, DISTINCT!
		}

		if err := rows.Err(); err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("rows iteration error: %v", err)})
			return
		}

		fmt.Println(fromuserId)

		for _, toUserId := range userIds {
			insertSQL := fmt.Sprintf(
				`INSERT INTO %s.message (to_user_id, from_user_id, subject, message)
             VALUES ($1, $2, $3, $4)`,
				h.Config.DbSchema,
			)
			_, err := tx.Exec(ctx, insertSQL, toUserId, fromuserId, req.Subject, req.Message)
			if err != nil {
				gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to insert message: %v", err)})
				return
			}
		}
	} else if req.Context == "user" {
		// TODO: Implement!
	} else {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "Unknwon context"})
		return
	}

	if err = tx.Commit(gc); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("failed to commit transaction: %v", err)})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"message": "Message sent successfully",
	})
}
