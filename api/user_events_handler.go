package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sndcds/uranus/app"
)

func AdminHandlerUserEvents(gc *gin.Context) {
	modeStr, _ := GetContextParam(gc, "mode")
	fmt.Println("query mode:", modeStr)
	//
	switch modeStr {
	case "organizer":
		authFetchEventsByOrganizer(gc)
		break
	case "venues":
		break
	case "space":
		break
	default:
		gc.JSON(http.StatusBadRequest, gin.H{
			"message": fmt.Sprintf("unknown mode: %s", modeStr),
		})
	}
}

func authFetchEventsByOrganizer(gc *gin.Context) {
	jsonData, httpStatus, err := authFetchEventsByOrganizerJSON(gc, app.Singleton.MainDbPool)
	if err != nil {
		gc.JSON(httpStatus, gin.H{"error": err.Error()})
		return
	}
	gc.Data(httpStatus, "application/json", jsonData)
}

func authFetchEventsByOrganizerJSON(gc *gin.Context, db *pgxpool.Pool) ([]byte, int, error) {
	ctx := gc.Request.Context()
	userId := gc.GetInt("user-id")

	orgId, ok := GetContextParameterAsInt(gc, "id")
	if !ok {
		return nil, http.StatusBadRequest, fmt.Errorf("error: variable id is required")
	}

	fmt.Println(app.Singleton.SqlQueryUserOrgEventsOverview)
	fmt.Println("userId", userId)
	fmt.Println("orgId", orgId)

	rows, err := db.Query(ctx, app.Singleton.SqlQueryUserOrgEventsOverview, userId, orgId, "2020-01-01", "2026-01-01")
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var results []map[string]interface{}

	for rows.Next() {
		var (
			eventOrgID      int
			eventOrgName    string
			venueID         int
			venueName       string
			venueOrgName    string
			spaceID         int
			spaceName       string
			eventID         int
			eventTitle      string
			eventDateID     int
			eventStart      *time.Time
			eventEnd        *time.Time
			eventCanEdit    bool
			eventCanDelete  bool
			eventCanRelease bool
		)

		var tempEventStart sql.NullTime
		var tempEventEnd sql.NullTime

		if err := rows.Scan(
			&eventOrgID,
			&eventOrgName,
			&venueID,
			&venueName,
			&venueOrgName,
			&spaceID,
			&spaceName,
			&eventID,
			&eventTitle,
			&eventDateID,
			&tempEventStart,
			&tempEventEnd,
			&eventCanEdit,
			&eventCanDelete,
			&eventCanRelease); err != nil {
			return nil, http.StatusInternalServerError, fmt.Errorf("row scan failed: %w", err)
		}

		if tempEventStart.Valid {
			eventStart = &tempEventStart.Time
		} else {
			eventStart = nil
		}

		if tempEventEnd.Valid {
			eventEnd = &tempEventEnd.Time
		} else {
			eventEnd = nil
		}

		item := map[string]interface{}{
			"event_org_id":      eventOrgID,
			"event_org_name":    eventOrgName,
			"venue_id":          venueID,
			"venue_name":        venueName,
			"venue_org_name":    venueOrgName,
			"space_id":          spaceID,
			"space_name":        spaceName,
			"event_id":          eventID,
			"event_title":       eventTitle,
			"event_date_id":     eventDateID,
			"event_start":       formatDate(eventStart),
			"event_end":         formatDate(eventEnd),
			"event_can_edit":    eventCanEdit,
			"event_can_delete":  eventCanDelete,
			"event_can_release": eventCanRelease,
		}

		results = append(results, item)
	}

	if err := rows.Err(); err != nil {
		return nil, http.StatusInternalServerError, err
	}

	jsonBytes, err := json.Marshal(results)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	// Print the JSON to the console
	fmt.Println(string(jsonBytes))

	return jsonBytes, http.StatusOK, nil
}

func formatDate(t *time.Time) string {
	if t == nil {
		return "" // or "null", or nil if using JSON encoding directly
	}
	return t.Format("02.01.2006 15:04")
}
