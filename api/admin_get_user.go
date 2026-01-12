package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// TODO: Code review
// TODO: Check permission to get user information

func (h *ApiHandler) AdminGetUser(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId, ok := ParamInt(gc, "userId")

	if !ok {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "userId required"})
		return
	}

	query := fmt.Sprintf(`
SELECT row_to_json(u) AS user FROM (
	SELECT user_name, email_address, COALESCE(display_name, first_name || ' ' || last_name) AS display_name
	FROM %s.user WHERE id = $1
) u`,
		h.Config.DbSchema)

	var userJSON []byte
	err := h.DbPool.QueryRow(ctx, query, userId).Scan(&userJSON)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			gc.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		} else {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	gc.Data(http.StatusOK, "application/json", userJSON)
}
