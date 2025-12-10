package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// TODO: Review code

type BitStatus struct {
	Bit     int  `json:"bit"`
	Enabled bool `json:"enabled"`
}

func (h *ApiHandler) AdminGetUserContextPermissions(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := h.DbPool

	userId, ok := ParamInt(gc, "userId")
	if !ok {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "user ID is required"})
		return
	}

	contextId, ok := ParamInt(gc, "contextId")
	if !ok {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "context ID is required"})
		return
	}

	contextName := gc.Param("contextName")
	if !isValidContext(contextName) {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "invalid context"})
		return
	}

	fmt.Println("userId:", userId, ", contextId:", contextId, ", contextName:", contextName)

	sql := fmt.Sprintf(`
		SELECT permissions FROM %s.user_%s_link WHERE user_id = $1 AND %s_id = $2`,
		h.Config.DbSchema, contextName, contextName)
	fmt.Println("sql_utils:", sql)

	var permissions int64
	err := pool.QueryRow(ctx, sql, userId, contextId).Scan(&permissions)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	gc.JSON(http.StatusOK, gin.H{"permissions": permissions})
}

func isValidContext(name string) bool {
	switch name {
	case "organizer", "venue", "space", "event":
		return true
	default:
		return false
	}
}
