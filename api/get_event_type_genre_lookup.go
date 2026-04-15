package api

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) GetEventTypeGenreLookup(gc *gin.Context) {
	ctx := gc.Request.Context()
	apiRequest := grains_api.NewRequest(gc, "event-type-genre-lookup")

	query := app.UranusInstance.SqlEventTypeGenreLookup
	rows, err := h.DbPool.Query(ctx, query)
	if err != nil {
		apiRequest.DatabaseError()
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
			apiRequest.DatabaseError()
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
		apiRequest.DatabaseError()
		return
	}

	apiRequest.Success(http.StatusOK, result, "")
}
