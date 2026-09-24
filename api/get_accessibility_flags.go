package api

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/model"

	"github.com/sndcds/grains/grains_api"
)

func (h *ApiHandler) GetAccessibilityFlags(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "get-accessibility-flags")

	ctx := gc.Request.Context()
	lang := gc.DefaultQuery("lang", "en")

	query := fmt.Sprintf(
		`SELECT
			flag AS id,
			topic_id,
			name
		FROM %s.accessibility_flag
		WHERE iso_639_1 = $1
		ORDER BY topic_id, flag`,
		h.DbSchema,
	)

	rows, err := h.DbPool.Query(ctx, query, lang)
	if err != nil {
		apiRequest.InternalServerError()
		return
	}
	defer rows.Close()

	accFlags := make([]model.AccessibilityFlag, 0)

	for rows.Next() {
		var flag model.AccessibilityFlag

		if err := rows.Scan(
			&flag.Id,
			&flag.TopicId,
			&flag.Name,
		); err != nil {
			apiRequest.InternalServerError()
			return
		}

		accFlags = append(accFlags, flag)
	}

	if err := rows.Err(); err != nil {
		apiRequest.InternalServerError()
		return
	}

	apiRequest.Success(200, accFlags)
}