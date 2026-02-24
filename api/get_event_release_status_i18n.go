package api

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
)

// EventReleaseStatusOption represents a single key/name pair
type EventReleaseStatusOption struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

// GetEventReleaseStatusI18n handles /api/event-release-status-i18n
func (h *ApiHandler) GetEventReleaseStatusI18n(gc *gin.Context) {
	ctx := gc.Request.Context()
	apiRequest := grains_api.NewRequest(gc, "event-release-status-i18n")

	// Get lang query param, default to "en"
	langParam := gc.DefaultQuery("lang", "en")
	langs := strings.Split(langParam, ",") // e.g., "de,da,en"
	apiRequest.SetMeta("languages", strings.Join(langs, ","))

	// Prepare placeholders for SQL: $1,$2,...
	args := make([]string, len(langs))
	placeholders := make([]string, len(langs))
	for i, l := range langs {
		args[i] = l
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}

	// SQL query: fetch all statuses + order for requested languages
	query := fmt.Sprintf(`
		SELECT iso_639_1, key, name, "order"
		FROM %s.event_release_status_i18n
		WHERE iso_639_1 IN (%s)
		ORDER BY iso_639_1, "order"`,
		h.DbSchema, strings.Join(placeholders, ","))

	// Convert []string to []interface{} for Query args
	queryArgs := make([]interface{}, len(args))
	for i, v := range args {
		queryArgs[i] = v
	}

	rows, err := h.DbPool.Query(ctx, query, queryArgs...)
	if err != nil {
		debugf("Error in GetEventReleaseStatusI18n: %s", err.Error())
		apiRequest.DatabaseError()
		return
	}
	defer rows.Close()

	// Build i18n map and order map
	i18n := make(map[string]map[string]string)
	orderMap := make(map[string]int) // key -> order

	for rows.Next() {
		var iso, key, name string
		var order int
		if err := rows.Scan(&iso, &key, &name, &order); err != nil {
			debugf("Error scanning row: %s", err.Error())
			apiRequest.DatabaseError()
			return
		}

		// Build i18n
		if _, ok := i18n[iso]; !ok {
			i18n[iso] = make(map[string]string)
		}
		i18n[iso][key] = name

		// Capture order (once per key, same across languages)
		if _, exists := orderMap[key]; !exists {
			orderMap[key] = order
		}
	}

	if err := rows.Err(); err != nil {
		debugf("Rows error: %s", err.Error())
		apiRequest.DatabaseError()
		return
	}

	// Generate ordered keys slice
	keyOrders := make([]struct {
		Key   string
		Order int
	}, 0, len(orderMap))
	for k, o := range orderMap {
		keyOrders = append(keyOrders, struct {
			Key   string
			Order int
		}{k, o})
	}
	sort.Slice(keyOrders, func(i, j int) bool {
		return keyOrders[i].Order < keyOrders[j].Order
	})
	orderSlice := make([]string, 0, len(keyOrders))
	for _, ko := range keyOrders {
		orderSlice = append(orderSlice, ko.Key)
	}

	// Return JSON using grains_api pattern
	response := map[string]interface{}{
		"i18n":  i18n,
		"order": orderSlice,
	}
	apiRequest.Success(http.StatusOK, response, "")
}
