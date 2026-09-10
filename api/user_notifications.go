package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
)

type UserNotification struct {
	UUID             string          `json:"uuid"`
	Type             string          `json:"type"`
	OrganizationUUID *string         `json:"organization_uuid"`
	ActionURL        *string         `json:"action_url"`
	Metadata         json.RawMessage `json:"metadata"`
	CreatedAt        time.Time       `json:"created_at"`
	ReadAt           *time.Time      `json:"read_at"`
	DismissedAt      *time.Time      `json:"dismissed_at"`
}

// Authentication comes exclusively from JWTMiddleware, never from request data.
func (h *ApiHandler) AdminGetNotifications(gc *gin.Context) {
	req := grains_api.NewRequest(gc, "admin-get-notifications")
	user := h.userUuid(gc)
	if _, err := uuid.Parse(user); err != nil {
		req.Error(http.StatusUnauthorized, "authentication required")
		return
	}
	status := gc.DefaultQuery("status", "active")
	if status != "active" && status != "unread" && status != "read" && status != "dismissed" {
		req.Error(http.StatusBadRequest, "invalid status")
		return
	}
	offset, err := strconv.Atoi(gc.DefaultQuery("offset", "0"))
	if err != nil || offset < 0 {
		req.Error(http.StatusBadRequest, "invalid offset")
		return
	}
	query := fmt.Sprintf(`SELECT uuid, type, organization_uuid, action_url, metadata, created_at, read_at, dismissed_at
 FROM %s.user_notification WHERE user_uuid = $1::uuid AND
 (($2 = 'dismissed' AND dismissed_at IS NOT NULL) OR
 (dismissed_at IS NULL AND ($2 = 'active' OR ($2 = 'unread' AND read_at IS NULL) OR ($2 = 'read' AND read_at IS NOT NULL))))
 ORDER BY created_at DESC, uuid DESC LIMIT 51 OFFSET $3`, h.DbSchema)
	rows, err := h.DbPool.Query(gc.Request.Context(), query, user, status, offset)
	if err != nil {
		req.InternalServerError()
		return
	}
	notifications, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (UserNotification, error) {
		var n UserNotification
		err := row.Scan(&n.UUID, &n.Type, &n.OrganizationUUID, &n.ActionURL, &n.Metadata, &n.CreatedAt, &n.ReadAt, &n.DismissedAt)
		return n, err
	})
	if err != nil {
		req.InternalServerError()
		return
	}
	if notifications == nil {
		notifications = []UserNotification{}
	}
	more := len(notifications) > 50
	if more {
		notifications = notifications[:50]
	}
	req.Success(http.StatusOK, gin.H{"notifications": notifications, "has_more": more})
}

func (h *ApiHandler) AdminReadNotification(gc *gin.Context)    { h.updateNotification(gc, false) }
func (h *ApiHandler) AdminDismissNotification(gc *gin.Context) { h.updateNotification(gc, true) }

func (h *ApiHandler) updateNotification(gc *gin.Context, dismiss bool) {
	req := grains_api.NewRequest(gc, "admin-update-notification")
	user := h.userUuid(gc)
	if _, err := uuid.Parse(user); err != nil {
		req.Error(http.StatusUnauthorized, "authentication required")
		return
	}
	id := gc.Param("notificationUuid")
	if _, err := uuid.Parse(id); err != nil {
		req.Error(http.StatusBadRequest, "invalid notification UUID")
		return
	}
	column := "read_at"
	if dismiss {
		column = "dismissed_at"
	}
	query := fmt.Sprintf(`UPDATE %s.user_notification SET %s = COALESCE(%s, now()) WHERE uuid = $1::uuid AND user_uuid = $2::uuid RETURNING read_at, dismissed_at`, h.DbSchema, column, column)
	var readAt, dismissedAt *time.Time
	err := h.DbPool.QueryRow(gc.Request.Context(), query, id, user).Scan(&readAt, &dismissedAt)
	if err == pgx.ErrNoRows {
		req.Error(http.StatusNotFound, "notification not found")
		return
	}
	if err != nil {
		req.InternalServerError()
		return
	}
	req.Success(http.StatusOK, gin.H{"read_at": readAt, "dismissed_at": dismissedAt})
}
