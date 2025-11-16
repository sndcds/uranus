package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

func (h *ApiHandler) AdminGetUser(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := h.DbPool

	userId, ok := ParamInt(gc, "userId")
	if !ok {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "user ID required"})
		return
	}

	// Query the user table
	sql := strings.Replace(`
		SELECT row_to_json(u) AS user
		FROM (
			SELECT
				user_name,
				email_address,
				COALESCE(display_name, first_name || ' ' || last_name) AS display_name
			FROM uranus.user
			WHERE id = 24
		) u`,
		"{{schema}}", h.Config.DbSchema, 1)

	var userJSON []byte
	err := pool.QueryRow(ctx, sql, userId).Scan(&userJSON)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			userJSON = []byte("{}")
		} else {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	gc.Data(http.StatusOK, "application/json", userJSON)
}
