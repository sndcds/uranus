package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	ics "github.com/arran4/golang-ical"
	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) GetEventDateICS(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "get-event-date-ics")
	ctx := gc.Request.Context()

	eventDateRequest, ok := h.ResolveEventDateRequest(gc, apiRequest)
	if !ok {
		apiRequest.InternalServerError()
		return
	}

	dateUuid := eventDateRequest.DateUuid

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

	err := h.DbPool.QueryRow(
		ctx,
		app.UranusInstance.SqlGetEventDateICS,
		dateUuid,
	).Scan(
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

	// ------------------------------------------------------------
	// Start / end
	// ------------------------------------------------------------

	dtStart := formatICSDatetime(startDate, startTime)

	if dtStart == "" {
		apiRequest.InternalServerError()
		return
	}

	dtEnd := ""

	// If an explicit end time exists, use it.
	// If no end date is supplied, assume the event ends on the start date.
	if endTime != "" {
		if endDate == "" {
			endDate = startDate
		}

		// If start and end are on the same date and the end time
		// is earlier than the start time, assume the event ends
		// after midnight on the following day.
		if endDate == startDate &&
			startTime != "" &&
			endTime < startTime {

			start, err := time.Parse("2006-01-02", endDate)
			if err == nil {
				endDate = start.AddDate(0, 0, 1).Format("2006-01-02")
			}
		}

		dtEnd = formatICSDatetime(endDate, endTime)
	}

	debugf("dtStart: %s, dtEnd: %s", dtStart, dtEnd)

	// ------------------------------------------------------------
	// Content
	// ------------------------------------------------------------

	title := str(event.Title)
	if title == "" {
		title = "Event"
	}

	description := str(event.Description)

	if subtitle := str(event.Subtitle); subtitle != "" {
		if description != "" {
			description = subtitle + "\n\n" + description
		} else {
			description = subtitle
		}
	}

	location := fmt.Sprintf(
		"%s, %s %s, %s",
		str(event.VenueName),
		str(event.VenueStreet),
		str(event.VenueHouseNumber),
		str(event.VenueCity),
	)

	// ------------------------------------------------------------
	// Calendar metadata
	// ------------------------------------------------------------

	uid := fmt.Sprintf(
		"%s@%s",
		event.EventDateUUID,
		h.Config.IcsDomain,
	)

	now := time.Now().UTC()

	// ------------------------------------------------------------
	// Build calendar
	// ------------------------------------------------------------

	cal, err := ics.NewCalendarWithOptions(
		ics.WithVersion("2.0"),
		ics.WithProductId("-//Uranus//EN"),
	)
	if err != nil {
		apiRequest.InternalServerError()
		return
	}

	cal.SetMethod(ics.MethodPublish)

	icalEvent := cal.AddEvent(uid)

	icalEvent.SetDtStampTime(now)

	// The original implementation uses floating local time.
	//
	// SetStartAt / SetEndAt serialize a time.Time. We therefore
	// construct the time without converting it to UTC so that the
	// local event time remains floating in the generated ICS.
	start, err := parseICSLocalTime(startDate, startTime)
	if err != nil {
		apiRequest.InternalServerError()
		return
	}

	icalEvent.SetStartAt(start)

	if dtEnd != "" {
		end, err := parseICSLocalTime(endDate, endTime)
		if err == nil {
			icalEvent.SetEndAt(end)
		}
	} else {
		// No explicit end time: default to one hour after start.
		icalEvent.SetEndAt(start.Add(time.Hour))
	}

	icalEvent.SetSummary(title)

	if description != "" {
		icalEvent.SetDescription(description)
	}

	if location != "" {
		icalEvent.SetLocation(location)
	}

	// SetOrganizer takes the email address separately from the
	// display name. The library handles serialization of the
	// ORGANIZER property and its CN parameter.
	if email := str(event.OrgContactEmail); email != "" {
		icalEvent.SetOrganizer(
			email,
			ics.WithCN(str(event.OrgName)),
		)
	}

	// RFC 5545 requires CRLF line endings for iCalendar content.
	icsContent := cal.Serialize(ics.WithNewLineWindows)

	// ------------------------------------------------------------
	// Response
	// ------------------------------------------------------------

	filename := title
	if filename == "" {
		filename = "event"
	}

	// Prevent the event title from becoming a header injection
	// vector through Content-Disposition.
	filename = sanitizeICSFilename(filename)

	gc.Header("Content-Type", "text/calendar; charset=utf-8")
	gc.Header(
		"Content-Disposition",
		fmt.Sprintf(`attachment; filename="%s.ics"`, filename),
	)

	gc.String(http.StatusOK, icsContent)
}

// formatICSDatetime parses date (YYYY-MM-DD) and time (HH:MM)
// and returns the floating ICS datetime representation.
//
// Example:
//
//	2026-09-19 + 18:30
//	=> 20260919T183000
func formatICSDatetime(dateStr, timeStr string) string {
	if dateStr == "" {
		return ""
	}

	if timeStr == "" {
		timeStr = "00:00"
	}

	t, err := time.Parse(
		"2006-01-02T15:04:05",
		fmt.Sprintf("%sT%s:00", dateStr, timeStr),
	)
	if err != nil {
		debugf("formatICSDatetime parse error: %s", err)
		return ""
	}

	return t.Format("20060102T150405")
}

// parseICSLocalTime parses an event date/time into a time.Time.
//
// UTC conversion is deliberately not performed here because the
// original ICS implementation uses floating local times.
func parseICSLocalTime(dateStr, timeStr string) (time.Time, error) {
	if dateStr == "" {
		return time.Time{}, fmt.Errorf("missing event date")
	}

	if timeStr == "" {
		timeStr = "00:00"
	}

	return time.Parse(
		"2006-01-02T15:04:05",
		fmt.Sprintf("%sT%s:00", dateStr, timeStr),
	)
}

// sanitizeICSFilename prevents CR/LF and quotes from entering
// the HTTP Content-Disposition header.
//
// This is separate from ICS escaping because this value is used
// in an HTTP header, not in an iCalendar property.
func sanitizeICSFilename(value string) string {
	value = strings.ReplaceAll(value, "\r", "")
	value = strings.ReplaceAll(value, "\n", "")
	value = strings.ReplaceAll(value, `"`, "'")

	if value == "" {
		return "event"
	}

	return value
}
