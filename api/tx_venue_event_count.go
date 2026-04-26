package api

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

func GetVenueEventDateCountTx(
	h *ApiHandler,
	gc *gin.Context,
	tx pgx.Tx,
	mode string,
	userUuid string,
	venueUuid string,
) (int, error) {
	ctx := gc.Request.Context()

	query := fmt.Sprintf(`
			SELECT
				COUNT(DISTINCT e.uuid) AS event_count
			FROM %s.event_date ed
			JOIN %s.event e ON e.uuid = ed.event_uuid
			JOIN %s.venue v ON v.uuid = COALESCE(ed.venue_uuid, e.venue_uuid)
			WHERE v.uuid = $1::uuid
		`, h.DbSchema, h.DbSchema, h.DbSchema)

	if mode == "upcoming" {
		query += fmt.Sprintf(" AND ed.start_date >= CURRENT_DATE")
	} else if mode == "past" {
		query += fmt.Sprintf(" AND ed.start_date < CURRENT_DATE")
	}

	query += `
		GROUP BY v.name, v.uuid
		`
	row := tx.QueryRow(ctx, query, venueUuid)

	var eventCount int
	err := row.Scan(&eventCount)
	if err != nil {
		return -1, err
	}

	return eventCount, nil
}
