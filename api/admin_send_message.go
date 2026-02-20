package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

// TODO: Code review

func (h *ApiHandler) AdminSendMessage(gc *gin.Context) {
	ctx := gc.Request.Context()
	fromUserId := h.userId(gc)

	if fromUserId <= 0 {
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

	tx, err := h.DbPool.Begin(ctx)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if req.Context == "organization" {
		organizationId := req.ContextId
		query := strings.Replace(
			`SELECT u.id, u.display_name
			FROM {{schema}}.user_organization_link ol
			JOIN {{schema}}.user u ON u.id = ol.user_id
			WHERE ol.organization_id = $1 AND (ol.permissions & $2) != 0`,
			"{{schema}}", h.DbSchema, -1)

		rows, err := tx.Query(ctx, query, organizationId, app.PermReceiveOrganizationMsgs)
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

		fmt.Println(fromUserId)

		for _, toUserId := range userIds {
			insertQuery := fmt.Sprintf(
				`INSERT INTO %s.message (to_user_id, from_user_id, subject, message)
             VALUES ($1, $2, $3, $4)`,
				h.DbSchema,
			)
			_, err := tx.Exec(ctx, insertQuery, toUserId, fromUserId, req.Subject, req.Message)
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
