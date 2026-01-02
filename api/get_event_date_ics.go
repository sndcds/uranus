package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/uranus/app"
)

// TODO: Review code

func (h *ApiHandler) GetEventDateICS(gc *gin.Context) {
	pool := h.DbPool
	ctx := gc.Request.Context()

	// Parse parameters
	eventId, ok := ParamInt(gc, "eventId")
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "event Id is required"})
		return
	}

	dateId, ok := ParamInt(gc, "dateId")
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "date Id is required"})
		return
	}

	langStr := gc.DefaultQuery("lang", "en")

	// Fetch event + date info from DB
	eventRow, err := pool.Query(ctx, app.UranusInstance.SqlGetEvent, eventId, gc.DefaultQuery("lang", "en"))
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer eventRow.Close()

	if !eventRow.Next() {
		gc.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
		return
	}

	eventData := mapRowToMap(eventRow)

	dateRows, err := pool.Query(ctx, app.UranusInstance.SqlGetEventDates, eventId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer dateRows.Close()

	var selectedDate map[string]interface{}
	for dateRows.Next() {
		d := mapRowToMap(dateRows)
		if intFromAny(d["event_date_id"]) == dateId {
			selectedDate = d
			break
		}
	}

	if selectedDate == nil {
		gc.JSON(http.StatusNotFound, gin.H{"error": "event date not found"})
		return
	}

	startDate, _ := selectedDate["start_date"].(string)
	startTime, _ := selectedDate["start_time"].(string)
	endDate, _ := selectedDate["end_date"].(string)
	endTime, _ := selectedDate["end_time"].(string)

	fmt.Println("endDate:", endDate)
	fmt.Println("endTime:", endTime)

	dtStart := formatICSDatetime(startDate, startTime)
	var dtEnd string
	if endDate != "" && endTime != "" {
		dtEnd = formatICSDatetime(endDate, endTime)
		// include DTEND in ICS
	} else {
		dtEnd = ""
		// skip DTEND in ICS
	}
	fmt.Println("dtEnd:", dtEnd)

	summary := fmt.Sprintf("%v", eventData["title"])
	var description string
	if subtitle, ok := eventData["subtitle"].(string); ok {
		description = subtitle
	} else {
		description = "" // fallback if nil or not a string
	}
	location := formatAddress(selectedDate)

	uid := fmt.Sprintf("%d-%d@%s", eventId, dateId, h.Config.ICSDomain)
	dtStamp := time.Now().UTC().Format("20060102T150405Z")

	prodId := fmt.Sprintf("-//Uranus//%s", strings.ToUpper(langStr))

	ics := strings.Join([]string{
		"BEGIN:VCALENDAR",
		"VERSION:2.0",
		"PRODID:" + prodId,
		"METHOD:PUBLISH",
		"BEGIN:VEVENT",
		"UID:" + uid,
		"DTSTAMP:" + dtStamp,
		"DTSTART:" + dtStart,
		"DTEND:" + dtEnd,
		"SUMMARY:" + escapeICSText(summary),
		"DESCRIPTION:" + escapeICSText(description),
		"LOCATION:" + location,
		"END:VEVENT",
		"END:VCALENDAR",
	}, "\r\n")
	ics = strings.Replace(ics, "DTEND:\r\n", "", 1)

	fmt.Println(ics)

	// Return as downloadable file
	gc.Header("Content-Type", "text/calendar")
	gc.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.ics"`, summary))
	gc.String(http.StatusOK, ics)
}

// formatICSDatetime parses date (YYYY-MM-DD) and time (HH:MM) strings and returns UTC ICS format
func formatICSDatetime(dateStr, timeStr string) string {
	if dateStr == "" {
		return ""
	}
	if timeStr == "" {
		timeStr = "00:00"
	}

	combined := fmt.Sprintf("%sT%s:00", dateStr, timeStr)

	// Load your local timezone (replace with your actual timezone)
	loc, err := time.LoadLocation("Europe/Berlin") // TODO: Handle Timezone individual
	if err != nil {
		loc = time.Local
	}

	// Parse using local timezone (this is the FIX)
	t, err := time.ParseInLocation("2006-01-02T15:04:05", combined, loc)
	if err != nil {
		fmt.Println("formatICSDatetime parse error:", err)
		return ""
	}

	// Convert to UTC for ICS
	return t.UTC().Format("20060102T150405Z")
}

// escapeICSText escapes commas, semicolons, and newlines for ICS
func escapeICSText(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, ",", "\\,")
	value = strings.ReplaceAll(value, ";", "\\;")
	value = strings.ReplaceAll(value, "\n", "\\n")
	return value
}

// Map a sql.Row to map[string]interface{}
func mapRowToMap(row pgx.Rows) map[string]interface{} {
	fieldDesc := row.FieldDescriptions()
	cols := make([]string, len(fieldDesc))
	for i, fd := range fieldDesc {
		cols[i] = string(fd.Name)
	}

	values, err := row.Values()
	if err != nil {
		return nil
	}

	data := make(map[string]interface{})
	for i, col := range cols {
		data[col] = values[i]
	}
	return data
}

func formatAddress(selectedDate map[string]interface{}) string {
	var addrStr string
	var streetStr string
	var cityStr string

	if selectedDate["venue_street"] != nil {
		streetStr = selectedDate["venue_street"].(string)
		if selectedDate["venue_house_number"] != nil {
			streetStr += " " + selectedDate["venue_house_number"].(string)
		}
	}

	if selectedDate["venue_city"] != nil {
		if selectedDate["venue_postal_code"] != nil {
			cityStr = selectedDate["venue_postal_code"].(string) + " "
		}
		cityStr += selectedDate["venue_city"].(string)
	}

	if selectedDate["venue_name"] != nil {
		addrStr = selectedDate["venue_name"].(string)
		if streetStr != "" {
			addrStr += "\\, " + streetStr
		}
		if cityStr != "" {
			addrStr += "\\, " + cityStr
		}
	}

	return addrStr
}
