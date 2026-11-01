package api

import (
	"context"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/sndcds/pluto"
	"github.com/sndcds/uranus/app"
)

type affectedQueries struct {
	EventUuids     string
	EventDateUuids string
}

var (
	queryOnce sync.Once
	queries   map[string]affectedQueries
)

func RefreshEventProjections(
	ctx context.Context,
	tx pgx.Tx,
	sourceTable string,
	uuids []string,
) error {
	if len(uuids) == 0 {
		return nil
	}

	initQueries()

	uuids = uniqueStrings(uuids)

	q, ok := queries[sourceTable]
	if !ok {
		debugf("unsupported source table: %s", sourceTable)
		return fmt.Errorf("unsupported source table: %s", sourceTable)
	}

	//--------------------------------------------------------------------------
	// Refresh events.
	//--------------------------------------------------------------------------

	if q.EventUuids != "" {

		eventUuids, err := fetchUuids(ctx, tx, q.EventUuids, uuids)
		if err != nil {
			return err
		}
		if len(eventUuids) > 0 {
			err := upsertEventProjection(ctx, tx, eventUuids)
			if err != nil {
				debugf("Error updating event projection: %v", err)
				return err
			}

			err = updateEventSearchVectors(ctx, tx, eventUuids)
			if err != nil {
				debugf("Error updating event search vectors: %v", err)
				return err
			}
		}
	}

	//--------------------------------------------------------------------------
	// Refresh event dates.
	//--------------------------------------------------------------------------

	if q.EventDateUuids != "" {

		eventDateUuids, err := fetchUuids(ctx, tx, q.EventDateUuids, uuids)
		if err != nil {
			return err
		}
		if len(eventDateUuids) > 0 {
			err := upsertEventDateProjection(ctx, tx, eventDateUuids)
			if err != nil {
				debugf("Error updating event date projection: %v", err)
				return err
			}

			err = updateEventDateSearchVectors(ctx, tx, eventDateUuids)
			if err != nil {
				debugf("Error updating event date search vectors: %v", err)
				return err
			}
		}
	}

	return nil
}

func RefreshEventProjectionsCallback(entity string, uuids []string) pluto.TxFunc {
	return func(ctx context.Context, tx pgx.Tx) error {
		return RefreshEventProjections(ctx, tx, entity, uuids)
	}
}

func upsertEventProjection(ctx context.Context, tx pgx.Tx, eventUuids []string) error {
	if len(eventUuids) == 0 {
		return nil
	}
	_, err := tx.Exec(ctx, app.UranusInstance.SqlEventProjectionUpsert, eventUuids)
	return err
}

func upsertEventDateProjection(ctx context.Context, tx pgx.Tx, eventDateUuids []string) error {
	if len(eventDateUuids) == 0 {
		return nil
	}
	_, err := tx.Exec(ctx, app.UranusInstance.SqlEventDateProjectionUpsert, eventDateUuids)
	return err
}

func initQueries() {
	queryOnce.Do(func() {
		schema := app.UranusInstance.Config.DbSchema
		queries = map[string]affectedQueries{
			"organization": {
				EventUuids: fmt.Sprintf(`
					SELECT uuid
					FROM %s.event
					WHERE org_uuid = ANY($1::uuid[])
				`, schema),

				EventDateUuids: fmt.Sprintf(`
					SELECT ed.uuid
					FROM %s.event_date ed
					JOIN %s.event e ON e.uuid = ed.event_uuid
					WHERE e.org_uuid = ANY($1::uuid[])
				`, schema, schema),
			},

			"venue": {
				EventUuids: fmt.Sprintf(`
					SELECT uuid
					FROM %s.event
					WHERE venue_uuid = ANY($1::uuid[])
				`, schema),

				EventDateUuids: fmt.Sprintf(`
					SELECT uuid
					FROM %s.event_date
					WHERE venue_uuid = ANY($1::uuid[])
				`, schema),
			},

			"space": {
				EventUuids: fmt.Sprintf(`
					SELECT uuid
					FROM %s.event
					WHERE space_uuid = ANY($1::uuid[])
				`, schema),

				EventDateUuids: fmt.Sprintf(`
					SELECT uuid
					FROM %s.event_date
					WHERE space_uuid = ANY($1::uuid[])
				`, schema),
			},

			"pluto_image": { // TODO: relation to pluto_image
				EventUuids: fmt.Sprintf(`
					SELECT uuid
					FROM %s.event
					WHERE image_ids[1] = ANY($1::uuid[])
				`, schema),

				EventDateUuids: "", // image does not affect event_date
			},

			"event": {
				EventUuids: fmt.Sprintf(`
					SELECT uuid
					FROM %s.event
					WHERE uuid = ANY($1::uuid[])
				`, schema),

				EventDateUuids: fmt.Sprintf(`
					SELECT uuid
					FROM %s.event_date
					WHERE event_uuid = ANY($1::uuid[])
				`, schema),
			},

			"event_date": {
				EventUuids: "", // event_date update only affects itself, not the parent event
				EventDateUuids: fmt.Sprintf(`
					SELECT uuid
					FROM %s.event_date
					WHERE uuid = ANY($1::uuid[])
				`, schema),
			},
		}
	})
}

func fetchUuids(
	ctx context.Context,
	tx pgx.Tx,
	query string,
	uuids []string,
) ([]string, error) {

	rows, err := tx.Query(ctx, query, uuids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []string
	for rows.Next() {
		var uuid *string
		err = rows.Scan(&uuid)
		if err != nil {
			return nil, err
		}

		if uuid != nil {
			result = append(result, *uuid)
		}
	}

	return result, nil
}

func uniqueStrings(uuids []string) []string {

	seen := make(map[string]struct{}, len(uuids))
	out := make([]string, 0, len(uuids))

	for _, id := range uuids {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}

	return out
}

func updateEventSearchVectors(
	ctx context.Context,
	tx pgx.Tx,
	eventUuids []string,
) error {

	if len(eventUuids) == 0 {
		return nil
	}

	_, err := tx.Exec(ctx, app.UranusInstance.SqlUpdateEventSearchVector, eventUuids)
	return err
}

func updateEventDateSearchVectors(
	ctx context.Context,
	tx pgx.Tx,
	eventDateUuids []string,
) error {

	if len(eventDateUuids) == 0 {
		return nil
	}

	_, err := tx.Exec(ctx, app.UranusInstance.SqlUpdateEventDateSearchVector, eventDateUuids)
	return err
}
