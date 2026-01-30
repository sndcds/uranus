package api

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) GetEventTypeGenreLookup(gc *gin.Context) {
	ctx := gc.Request.Context()
	responseType := "event-type-genre-lookup"

	query := app.UranusInstance.SqlTypeGenreLookup
	rows, err := h.DbPool.Query(ctx, query)
	if err != nil {
		JSONDatabaseError(gc, responseType)
		return
	}
	defer rows.Close()

	type LanguageLookup struct {
		Types json.RawMessage `json:"types"`
	}

	result := map[string]LanguageLookup{}
	for rows.Next() {
		var (
			lang  string
			types json.RawMessage
		)

		if err := rows.Scan(&lang, &types); err != nil {
			JSONDatabaseError(gc, responseType)
			return
		}

		// Defensive: ensure non-null JSON
		if types == nil {
			types = json.RawMessage(`{}`)
		}

		result[lang] = LanguageLookup{
			Types: types,
		}
	}

	if err := rows.Err(); err != nil {
		JSONDatabaseError(gc, responseType)
		return
	}

	JSONSuccess(gc, responseType, result, nil)
}
