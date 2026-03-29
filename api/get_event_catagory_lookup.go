package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"github.com/sndcds/grains/grains_api"
)

func (h *ApiHandler) GetEventCategoryLookup(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "event-category-lookup")

	type category struct {
		Name string `json:"name"`
	}

	lang := gc.DefaultQuery("lang", "")
	var languages []string
	if lang != "" {
		for _, l := range strings.Split(lang, ",") {
			languages = append(languages, strings.TrimSpace(l))
		}
	}

	query :=
		`SELECT category_id, iso_639_1, name
		FROM ` + h.DbSchema + `.event_category
		WHERE ($1::text[] IS NULL OR iso_639_1 = ANY($1))
		ORDER BY iso_639_1, category_id`

	rows, err := h.DbPool.Query(gc, query, pq.Array(languages))
	if err != nil {
		apiRequest.Error(http.StatusInternalServerError, "database query failed: "+err.Error())
		return
	}
	defer rows.Close()

	// build nested structure: lang -> category_id -> category{name}
	filteredData := make(map[string]map[string]category)

	for rows.Next() {
		var id int
		var lang, name string
		if err := rows.Scan(&id, &lang, &name); err != nil {
			apiRequest.Error(http.StatusInternalServerError, "failed to scan row: "+err.Error())
			return
		}

		if _, ok := filteredData[lang]; !ok {
			filteredData[lang] = make(map[string]category)
		}

		filteredData[lang][strconv.Itoa(id)] = category{
			Name: name,
		}
	}

	if err := rows.Err(); err != nil {
		apiRequest.Error(http.StatusInternalServerError, "row iteration error: "+err.Error())
		return
	}

	// pass Go object to Success
	apiRequest.Success(http.StatusOK, filteredData, "")
}
