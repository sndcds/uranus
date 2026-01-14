package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// TODO: Review code

func (h *ApiHandler) GetAccessibilityFlags(gc *gin.Context) {
	ctx := gc.Request.Context()

	lang := gc.DefaultQuery("lang", "en")

	query := fmt.Sprintf(
		`SELECT flag AS id, topic_id, name FROM %s.accessibility_flags WHERE iso_639_1 = $1 ORDER BY topic_id, flag`,
		h.Config.DbSchema,
	)

	rows, err := h.DbPool.Query(ctx, query, lang)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type AccFlag struct {
		Id      *string `json:"id"`
		TopicId *string `json:"topic_id"`
		Name    *string `json:"name"`
	}

	var accFlags []AccFlag

	for rows.Next() {
		var flag AccFlag
		if err := rows.Scan(
			&flag.Id,
			&flag.TopicId,
			&flag.Name,
		); err != nil {
			fmt.Println(err.Error())
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		accFlags = append(accFlags, flag)
	}

	if err := rows.Err(); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(accFlags) == 0 {
		gc.JSON(http.StatusOK, []AccFlag{})
		return
	}

	gc.JSON(http.StatusOK, accFlags)
}
