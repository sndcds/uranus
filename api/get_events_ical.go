package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) GetEventsICS(gc *gin.Context) {
	ctx := gc.Request.Context()

	dateConditions, conditionsStr, limitClause, args, _, err := h.buildEventFilters(gc)
	if err != nil {
		gc.String(http.StatusBadRequest, err.Error())
		return
	}

	query := app.UranusInstance.SqlGetEventsProjected
	query = strings.Replace(query, "{{date_conditions}}", dateConditions, 1)
	query = strings.Replace(query, "{{conditions}}", conditionsStr, 1)
	query = strings.Replace(query, "{{limit}}", limitClause, 1)

	rows, err := h.DbPool.Query(ctx, query, args...)
	if err != nil {
		debugf("GetEventsICS query error: %v", err)
		gc.Status(http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var events []eventResponse

	for rows.Next() {
		var e eventResponse
		var typesJSON []byte

		err := rows.Scan(
			&e.DateUuid,
			&e.Uuid,
			&e.OrgUuid,
			&e.VenueUuid,
			&e.SpaceUuid,
			&e.StartDate,
			&e.StartTime,
			&e.EndDate,
			&e.EndTime,
			&e.EntryTime,
			&e.Duration,
			&e.AllDay,
			&e.ReleaseStatus,
			&e.TicketLink,
			&e.Title,
			&e.Subtitle,
			&e.Categories,
			&typesJSON,
			&e.Languages,
			&e.Tags,
			&e.OrgName,
			&e.ImageUuid,
			&e.VenueName,
			&e.VenueCity,
			&e.VenueStreet,
			&e.VenueHouse,
			&e.VenuePostal,
			&e.VenueState,
			&e.VenueCountry,
			&e.VenueLat,
			&e.VenueLon,
			&e.SpaceName,
			&e.SpaceAccessibilityFlags,
			&e.MinAge,
			&e.MaxAge,
			&e.VisitorInfoFlags,
		)
		if err != nil {
			debugf("GetEventsICS scan error: %v", err)
			gc.Status(http.StatusInternalServerError)
			return
		}

		var rawTypes [][]int
		if len(typesJSON) > 0 {
			if err := json.Unmarshal(typesJSON, &rawTypes); err != nil {
				debugf("GetEventsICS unmarshal error: %v", err)
				gc.Status(http.StatusInternalServerError)
				return
			}

			e.EventTypes = make([]eventType, len(rawTypes))
			for i, pair := range rawTypes {
				if len(pair) < 2 {
					continue
				}
				e.EventTypes[i] = eventType{
					TypeId:  pair[0],
					GenreId: pair[1],
				}
			}
		} else {
			e.EventTypes = []eventType{}
		}

		if e.ImageUuid != nil {
			path := ImageUrl(*e.ImageUuid)
			e.ImagePath = &path
		}

		events = append(events, e)
	}

	if err := rows.Err(); err != nil {
		debugf("GetEventsICS rows error: %v", err)
		gc.Status(http.StatusInternalServerError)
		return
	}

	cal, err := buildEventsICS(events, gc.Request.Host)
	if err != nil {
		debugf("GetEventsICS build error: %v", err)
		gc.Status(http.StatusInternalServerError)
		return
	}

	gc.Header("Content-Type", "text/calendar; charset=utf-8")
	gc.Header("Content-Disposition", `inline; filename="events.ics"`)
	gc.Header("Cache-Control", "public, max-age=300")
	gc.String(http.StatusOK, cal)
}

func buildEventsICS(events []eventResponse, host string) (string, error) {
	if host == "" {
		host = "localhost"
	}

	var b strings.Builder
	writeICSLine := func(s string) {
		b.WriteString(foldICSLine(s))
		b.WriteString("\r\n")
	}

	nowUTC := time.Now().UTC().Format("20060102T150405Z")

	writeICSLine("BEGIN:VCALENDAR")
	writeICSLine("PRODID:-//Uranus//Events Feed//DE")
	writeICSLine("VERSION:2.0")
	writeICSLine("CALSCALE:GREGORIAN")
	writeICSLine("METHOD:PUBLISH")
	writeICSLine("X-WR-CALNAME:Uranus Events")
	writeICSLine("X-WR-TIMEZONE:Europe/Berlin")

	writeEuropeBerlinVTimezone(&b)

	for _, e := range events {
		start, end, allDay, err := deriveEventTimes(e)
		if err != nil {
			debugf("Skipping event %d due to time parse error: %v", e.DateUuid, err)
			continue
		}

		uid := buildEventUID(e, host)
		description := buildEventDescription(e)
		location := buildEventLocation(e)
		url := stringPtrValue(e.TicketLink)
		lastModified := nowUTC

		writeICSLine("BEGIN:VEVENT")
		writeICSLine("UID:" + escapeICS(uid))
		writeICSLine("DTSTAMP:" + nowUTC)
		writeICSLine("LAST-MODIFIED:" + lastModified)

		if allDay {
			writeICSLine("DTSTART;VALUE=DATE:" + start.Format("20060102"))
			writeICSLine("DTEND;VALUE=DATE:" + end.Format("20060102"))
		} else {
			writeICSLine("DTSTART;TZID=Europe/Berlin:" + start.Format("20060102T150405"))
			writeICSLine("DTEND;TZID=Europe/Berlin:" + end.Format("20060102T150405"))
		}

		writeICSLine("SUMMARY:" + escapeICS(strings.TrimSpace(e.Title)))

		if description != "" {
			writeICSLine("DESCRIPTION:" + escapeICS(description))
		}
		if location != "" {
			writeICSLine("LOCATION:" + escapeICS(location))
		}
		if url != "" {
			writeICSLine("URL:" + escapeICS(url))
		}

		if e.VenueLat != nil && e.VenueLon != nil {
			writeICSLine("GEO:" + formatFloat(*e.VenueLat) + ";" + formatFloat(*e.VenueLon))
		}

		writeICSLine("STATUS:CONFIRMED")
		writeICSLine("TRANSP:OPAQUE")
		writeICSLine("END:VEVENT")
	}

	writeICSLine("END:VCALENDAR")
	return b.String(), nil
}

func deriveEventTimes(e eventResponse) (start time.Time, end time.Time, allDay bool, err error) {
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		return time.Time{}, time.Time{}, false, err
	}

	allDay = false
	if e.AllDay != nil {
		allDay = *e.AllDay
	}

	if allDay {
		start, err = time.ParseInLocation("2006-01-02", e.StartDate, loc)
		if err != nil {
			return time.Time{}, time.Time{}, true, fmt.Errorf("invalid all-day start_date %q: %w", e.StartDate, err)
		}

		if e.EndDate != nil && strings.TrimSpace(*e.EndDate) != "" {
			parsedEnd, parseErr := time.ParseInLocation("2006-01-02", strings.TrimSpace(*e.EndDate), loc)
			if parseErr != nil {
				return time.Time{}, time.Time{}, true, fmt.Errorf("invalid all-day end_date %q: %w", *e.EndDate, parseErr)
			}
			return start, parsedEnd.AddDate(0, 0, 1), true, nil
		}

		return start, start.AddDate(0, 0, 1), true, nil
	}

	start, err = parseLocalDateTime(e.StartDate, e.StartTime, loc)
	if err != nil {
		return time.Time{}, time.Time{}, false, fmt.Errorf("invalid start datetime: %w", err)
	}

	if e.EndDate != nil && strings.TrimSpace(*e.EndDate) != "" && e.EndTime != nil && strings.TrimSpace(*e.EndTime) != "" {
		end, err = parseLocalDateTime(*e.EndDate, *e.EndTime, loc)
		if err != nil {
			return time.Time{}, time.Time{}, false, fmt.Errorf("invalid end datetime: %w", err)
		}
		return start, ensureEndAfterStart(start, end), false, nil
	}

	if e.EndTime != nil && strings.TrimSpace(*e.EndTime) != "" {
		end, err = parseLocalDateTime(e.StartDate, *e.EndTime, loc)
		if err != nil {
			return time.Time{}, time.Time{}, false, fmt.Errorf("invalid same-day end_time: %w", err)
		}

		if !end.After(start) {
			if e.EndDate != nil && strings.TrimSpace(*e.EndDate) != "" {
				end, err = parseLocalDateTime(*e.EndDate, *e.EndTime, loc)
				if err == nil && end.After(start) {
					return start, end, false, nil
				}
			}
			end = end.Add(24 * time.Hour)
		}

		return start, end, false, nil
	}

	if e.Duration != nil && *e.Duration > 0 {
		return start, start.Add(time.Duration(*e.Duration) * time.Minute), false, nil
	}

	return start, start.Add(2 * time.Hour), false, nil
}

func parseLocalDateTime(dateStr, timeStr string, loc *time.Location) (time.Time, error) {
	dateStr = strings.TrimSpace(dateStr)
	timeStr = strings.TrimSpace(timeStr)

	if dateStr == "" || timeStr == "" {
		return time.Time{}, fmt.Errorf("missing date or time")
	}

	timeStr = normalizeClockTime(timeStr)

	layouts := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
	}

	var lastErr error
	value := dateStr + " " + timeStr
	for _, layout := range layouts {
		t, err := time.ParseInLocation(layout, value, loc)
		if err == nil {
			return t, nil
		}
		lastErr = err
	}

	return time.Time{}, lastErr
}

func ensureEndAfterStart(start, end time.Time) time.Time {
	if end.After(start) {
		return end
	}
	return start.Add(2 * time.Hour)
}

func buildEventUID(e eventResponse, host string) string {
	return fmt.Sprintf("eventdate-%d@%s", e.DateUuid, sanitizeHost(host))
}

func buildEventDescription(e eventResponse) string {
	var parts []string

	if s := stringPtrValue(e.Subtitle); s != "" {
		parts = append(parts, s)
	}

	parts = append(parts, "Veranstalter: "+e.OrgName)

	if s := stringPtrValue(e.SpaceName); s != "" {
		parts = append(parts, "Raum: "+s)
	}
	if e.MinAge != nil {
		parts = append(parts, "Mindestalter: "+strconv.Itoa(*e.MinAge))
	}
	if e.MaxAge != nil {
		parts = append(parts, "Höchstalter: "+strconv.Itoa(*e.MaxAge))
	}
	if s := stringPtrValue(e.ImagePath); s != "" {
		parts = append(parts, "Bild: "+s)
	}

	return strings.Join(parts, "\n")
}

func buildEventLocation(e eventResponse) string {
	var parts []string

	if s := stringPtrValue(e.VenueName); s != "" {
		parts = append(parts, s)
	}

	street := strings.TrimSpace(strings.Join([]string{
		stringPtrValue(e.VenueStreet),
		stringPtrValue(e.VenueHouse),
	}, " "))
	if street != "" {
		parts = append(parts, street)
	}

	cityLine := strings.TrimSpace(strings.Join([]string{
		stringPtrValue(e.VenuePostal),
		stringPtrValue(e.VenueCity),
	}, " "))
	if cityLine != "" {
		parts = append(parts, cityLine)
	}

	if s := stringPtrValue(e.VenueState); s != "" {
		parts = append(parts, s)
	}
	if s := stringPtrValue(e.VenueCountry); s != "" {
		parts = append(parts, s)
	}

	return strings.Join(parts, ", ")
}

func writeEuropeBerlinVTimezone(b *strings.Builder) {
	lines := []string{
		"BEGIN:VTIMEZONE",
		"TZID:Europe/Berlin",
		"X-LIC-LOCATION:Europe/Berlin",
		"BEGIN:DAYLIGHT",
		"TZOFFSETFROM:+0100",
		"TZOFFSETTO:+0200",
		"TZNAME:CEST",
		"DTSTART:19700329T020000",
		"RRULE:FREQ=YEARLY;BYMONTH=3;BYDAY=-1SU",
		"END:DAYLIGHT",
		"BEGIN:STANDARD",
		"TZOFFSETFROM:+0200",
		"TZOFFSETTO:+0100",
		"TZNAME:CET",
		"DTSTART:19701025T030000",
		"RRULE:FREQ=YEARLY;BYMONTH=10;BYDAY=-1SU",
		"END:STANDARD",
		"END:VTIMEZONE",
	}

	for _, line := range lines {
		b.WriteString(foldICSLine(line))
		b.WriteString("\r\n")
	}
}

func escapeICS(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, ";", `\;`)
	s = strings.ReplaceAll(s, ",", `\,`)
	s = strings.ReplaceAll(s, "\r\n", `\n`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\n`)
	return s
}

func foldICSLine(s string) string {
	const max = 75

	if len(s) <= max {
		return s
	}

	var out strings.Builder
	for len(s) > max {
		out.WriteString(s[:max])
		out.WriteString("\r\n ")
		s = s[max:]
	}
	out.WriteString(s)

	return out.String()
}

func normalizeClockTime(s string) string {
	s = strings.TrimSpace(s)

	if len(s) == 5 {
		return s + ":00"
	}
	return s
}

func sanitizeHost(host string) string {
	host = strings.TrimSpace(host)
	host = strings.ReplaceAll(host, ":", "-")
	if host == "" {
		return "localhost"
	}
	return host
}

func stringPtrValue(s *string) string {
	if s == nil {
		return ""
	}
	return strings.TrimSpace(*s)
}

func formatFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', 6, 64)
}
