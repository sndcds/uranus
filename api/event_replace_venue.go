package api

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/sndcds/uranus/app"
)

type ReplaceVenueUuidResult struct {
	EventsUpdated     int64    `json:"events_updated"`
	EventDatesUpdated int64    `json:"event_dates_updated"`
	EventUuids        []string `json:"event_uuids"`
	EventDateUuids    []string `json:"event_date_uuids"`
	DryRun            bool     `json:"dry_run"`
}

func ReplaceVenueUuid(
	ctx context.Context,
	tx pgx.Tx,
	oldVenueUuid string,
	newVenueUuid string,
	dryRun bool,
) (*ReplaceVenueUuidResult, error) {
	schema := app.UranusInstance.Config.DbSchema

	if oldVenueUuid == "" || newVenueUuid == "" {
		return nil, fmt.Errorf("venue UUIDs must not be empty")
	}

	if oldVenueUuid == newVenueUuid {
		return nil, fmt.Errorf("old and new venue UUIDs must differ")
	}

	result := &ReplaceVenueUuidResult{
		DryRun: dryRun,
	}

	// Verify the destination venue exists.
	var venueExists bool

	err := tx.QueryRow(ctx, fmt.Sprintf(`
		SELECT EXISTS (
			SELECT 1
			FROM %s.venue
			WHERE uuid = $1::uuid
		)
	`, schema), newVenueUuid).Scan(&venueExists)
	if err != nil {
		return nil, fmt.Errorf("check destination venue: %w", err)
	}

	if !venueExists {
		return nil, fmt.Errorf("destination venue %q does not exist", newVenueUuid)
	}

	// Collect affected events.
	result.EventUuids, err = fetchVenueUuids(
		ctx,
		tx,
		fmt.Sprintf(`
			SELECT uuid
			FROM %s.event
			WHERE venue_uuid = $1::uuid
			ORDER BY uuid
		`, schema),
		oldVenueUuid,
	)
	if err != nil {
		return nil, fmt.Errorf("fetch affected events: %w", err)
	}

	// Collect every date whose projection will be refreshed:
	// 1. Dates explicitly assigned to the old venue.
	// 2. All dates belonging to affected events, including inherited venues.
	result.EventDateUuids, err = fetchEventDateUuids(
		ctx,
		tx,
		fmt.Sprintf(`
			SELECT ed.uuid
			FROM %s.event_date ed
			WHERE ed.venue_uuid = $1::uuid
			   OR ed.event_uuid = ANY($2::uuid[])
			ORDER BY ed.uuid
		`, schema),
		oldVenueUuid,
		result.EventUuids,
	)
	if err != nil {
		return nil, fmt.Errorf("fetch affected event dates: %w", err)
	}

	// Explicitly affected dates that will NOT be refreshed as part
	// of an affected event's projection refresh.
	standaloneDateUuids, err := fetchEventDateUuids(
		ctx,
		tx,
		fmt.Sprintf(`
			SELECT ed.uuid
			FROM %s.event_date ed
			WHERE ed.venue_uuid = $1::uuid
			  AND (
			      ed.event_uuid IS NULL
			      OR NOT (ed.event_uuid = ANY($2::uuid[]))
			  )
			ORDER BY ed.uuid
		`, schema),
		oldVenueUuid,
		result.EventUuids,
	)
	if err != nil {
		return nil, fmt.Errorf("fetch standalone event dates: %w", err)
	}

	// Dry run: return the complete preview without modifying anything.
	if dryRun {
		return result, nil
	}

	// Update source events.
	tag, err := tx.Exec(ctx, fmt.Sprintf(`
		UPDATE %s.event
		SET venue_uuid = $1::uuid,
		    space_uuid = NULL
		WHERE venue_uuid = $2::uuid
	`, schema), newVenueUuid, oldVenueUuid)
	if err != nil {
		return nil, fmt.Errorf("update events: %w", err)
	}
	result.EventsUpdated = tag.RowsAffected()

	// Update source event dates with explicit venue overrides.
	tag, err = tx.Exec(ctx, fmt.Sprintf(`
		UPDATE %s.event_date
		SET venue_uuid = $1::uuid,
		    space_uuid = NULL
		WHERE venue_uuid = $2::uuid
	`, schema), newVenueUuid, oldVenueUuid)
	if err != nil {
		return nil, fmt.Errorf("update event dates: %w", err)
	}
	result.EventDatesUpdated = tag.RowsAffected()

	// Refresh affected events and all their event-date projections.
	if len(result.EventUuids) > 0 {
		if err := RefreshEventProjections(
			ctx, tx, "event", result.EventUuids,
		); err != nil {
			return nil, fmt.Errorf("refresh event projections: %w", err)
		}
	}

	// Refresh explicitly affected dates not already covered above.
	if len(standaloneDateUuids) > 0 {
		if err := RefreshEventProjections(
			ctx, tx, "event_date", standaloneDateUuids,
		); err != nil {
			return nil, fmt.Errorf("refresh standalone event-date projections: %w", err)
		}
	}

	return result, nil
}

func fetchVenueUuids(
	ctx context.Context,
	tx pgx.Tx,
	query string,
	venueUuid string,
) ([]string, error) {
	rows, err := tx.Query(ctx, query, venueUuid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var uuids []string

	for rows.Next() {
		var uuid string
		if err := rows.Scan(&uuid); err != nil {
			return nil, err
		}
		uuids = append(uuids, uuid)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return uuids, nil
}

func fetchEventDateUuids(
	ctx context.Context,
	tx pgx.Tx,
	query string,
	venueUuid string,
	eventUuids []string,
) ([]string, error) {
	rows, err := tx.Query(ctx, query, venueUuid, eventUuids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var uuids []string

	for rows.Next() {
		var uuid string
		if err := rows.Scan(&uuid); err != nil {
			return nil, err
		}
		uuids = append(uuids, uuid)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return uuids, nil
}
