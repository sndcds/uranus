package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) GetEventDateICS(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "get-event-date-ics")
	ctx := gc.Request.Context()

	dateUuid := gc.Param("dateUuid")
	if dateUuid == "" {
		apiRequest.Required("dateUuid is required")
		return
	}

	lang := gc.DefaultQuery("lang", "en")
	fmt.Println(lang)

	type EventDateICS struct {
		EventDateUUID    string
		VenueName        *string
		VenueStreet      *string
		VenueHouseNumber *string
		VenueCity        *string
		StartDate        *string
		StartTime        *string
		EndDate          *string
		EndTime          *string
		Title            *string
		Subtitle         *string
		Description      *string
		OrgName          *string
		OrgContactEmail  *string
	}

	var event EventDateICS
	err := h.DbPool.QueryRow(ctx, app.UranusInstance.SqlGetEventDateICS, dateUuid).Scan(
		&event.EventDateUUID,
		&event.VenueName,
		&event.VenueStreet,
		&event.VenueHouseNumber,
		&event.VenueCity,
		&event.StartDate,
		&event.StartTime,
		&event.EndDate,
		&event.EndTime,
		&event.Title,
		&event.Subtitle,
		&event.Description,
		&event.OrgName,
		&event.OrgContactEmail,
	)
	if err != nil {
		apiRequest.InternalServerError()
		return
	}

	str := func(p *string) string {
		if p == nil {
			return ""
		}
		return *p
	}

	startDate := str(event.StartDate)
	startTime := str(event.StartTime)
	endDate := str(event.EndDate)
	endTime := str(event.EndTime)

	dtStart := formatICSDatetime(startDate, startTime)

	dtEnd := ""
	if endDate != "" && endTime != "" {
		dtEnd = formatICSDatetime(endDate, endTime)
	}

	debugf("dtStart: %s, dtEnd: %s", dtStart, dtEnd)

	// Content

	title := str(event.Title)
	if title == "" {
		title = "Event"
	}

	description := str(event.Description)
	if sub := str(event.Subtitle); sub != "" {
		description = sub + "\n\n" + description
	}

	location := fmt.Sprintf("%s, %s %s, %s",
		str(event.VenueName),
		str(event.VenueStreet),
		str(event.VenueHouseNumber),
		str(event.VenueCity),
	)

	// ICS fields

	// Floating time format
	timeFormat := "20060102T150405"

	uid := fmt.Sprintf("%s@%s", event.EventDateUUID, h.Config.IcsDomain)
	dtStamp := time.Now().UTC().Format(timeFormat)

	// Build ICS

	var b strings.Builder

	b.WriteString("BEGIN:VCALENDAR\r\n")
	b.WriteString("VERSION:2.0\r\n")
	b.WriteString("PRODID:-//Uranus//EN\r\n")
	b.WriteString("METHOD:PUBLISH\r\n")

	b.WriteString("BEGIN:VEVENT\r\n")
	b.WriteString("UID:" + uid + "\r\n")
	b.WriteString("DTSTAMP:" + dtStamp + "\r\n")
	b.WriteString("DTSTART:" + dtStart + "\r\n")

	start, err := time.Parse(timeFormat, dtStart)
	if err == nil {
		dtEnd = start.Add(time.Hour).Format(timeFormat)
		b.WriteString("DTEND:" + dtEnd + "\r\n")
	} else {
		debugf(err.Error())
	}

	b.WriteString("SUMMARY:" + escapeICSText(title) + "\r\n")
	b.WriteString("DESCRIPTION:" + escapeICSText(description) + "\r\n")
	b.WriteString("LOCATION:" + escapeICSText(location) + "\r\n")

	if email := str(event.OrgContactEmail); email != "" {
		b.WriteString("ORGANIZER;CN=" + escapeICSText(str(event.OrgName)) +
			":mailto:" + email + "\r\n")
	}

	b.WriteString("END:VEVENT\r\n")
	b.WriteString("END:VCALENDAR\r\n")

	ics := b.String()

	// Response

	filename := title
	if filename == "" {
		filename = "event"
	}

	gc.Header("Content-Type", "text/calendar; charset=utf-8")
	gc.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.ics"`, filename))
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
	return t.Format("20060102T150405")
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
