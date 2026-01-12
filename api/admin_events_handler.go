package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// TODO: Review code

func AdminEventsHandler(gc *gin.Context) {
	modeStr, _ := GetContextParam(gc, "mode")

	fmt.Println("modeStr", modeStr)

	switch modeStr {
	case "table-view":
		fetchTableView(gc)
		break

	default:
		gc.JSON(http.StatusBadRequest, gin.H{
			"message": fmt.Sprintf("unknown mode: %s", modeStr),
		})
	}
}

func fetchTableView(gc *gin.Context) {
	/*
		db := app.Singleton.MainDbPool
		ctx := gc.Request.Context()
		userId := h.userId(gc);

		orgID, ok := GetContextParameterAsInt(gc, "org-id")
		if !ok {
			gc.JSON(http.StatusBadRequest, gin.H{"message": "variable org-id is required"})
			return
		}

		sql_utils := app.Singleton.SqlQueryUserOrgEventsOverview
		rows, err := db.Query(ctx, sql_utils, userId, orgID, "2020-01-01", "3000-01-01")
		if err != nil {
			gc.JSON(http.StatusInternalServerError, err)
			return
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
				// eventStart      *time.Time
				// eventEnd        *time.Time
				eventCanEdit    bool
				eventCanDelete  bool
				eventCanRelease bool
			)

			var tempEventStart pgtype.Timestamp
			var tempEventEnd pgtype.Timestamp

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
				gc.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
				return
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
				"event_org_id":   eventOrgID,
				"event_org_name": eventOrgName,
				"venue_id":       venueID,
				"venue_name":     venueName,
				"venue_org_name": venueOrgName,
				"space_id":       spaceID,
				"space_name":     spaceName,
				"event_id":       eventID,
				"event_title":    eventTitle,
				"event_date_id":  eventDateID,
				// "event_start":       formatDate(eventStart),
				// "event_end":         formatDate(eventEnd),
				"event_can_edit":    eventCanEdit,
				"event_can_delete":  eventCanDelete,
				"event_can_release": eventCanRelease,
			}

			results = append(results, item)
		}

		if err := rows.Err(); err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		}

		gc.JSON(http.StatusOK, results)
	*/
}
