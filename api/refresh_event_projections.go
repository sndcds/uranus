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

var (
	initProjectionQueries        sync.Once
	eventProjectionUpsertSql     string
	eventDateProjectionUpsertSql string
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

	fmt.Println("uuids:", uuids)

	q, ok := queries[sourceTable]
	if !ok {
		return fmt.Errorf("unsupported source table: %s", sourceTable)
	}

	// Refresh events
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
		}
		fmt.Println("eventUuids:", eventUuids)
	}

	// Refresh event dates
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
		}
		fmt.Println("eventDateUuids:", eventDateUuids)
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
	initProjectionSql()
	_, err := tx.Exec(ctx, eventProjectionUpsertSql, eventUuids)
	return err
}

func upsertEventDateProjection(ctx context.Context, tx pgx.Tx, eventDateUuids []string) error {
	if len(eventDateUuids) == 0 {
		return nil
	}
	initProjectionSql()
	_, err := tx.Exec(ctx, eventDateProjectionUpsertSql, eventDateUuids)
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

func initProjectionSql() {
	initProjectionQueries.Do(func() {
		eventProjectionUpsertSql = fmt.Sprintf(`
INSERT INTO %[1]s.event_projection (
    event_uuid, org_uuid, venue_uuid, space_uuid, release_status,
    title, subtitle, description, summary, image_uuid, languages, tags, categories, types,
    source_link, online_link, occasion_type_id, max_attendees, min_age, max_age,
    participation_info, meeting_point, ticket_flags,
	price_type, currency, min_price, max_price, visitor_info_flags,
    external_id, custom, style, search_text,
    org_name, org_contact_email, org_contact_phone, org_link,
    venue_name, venue_street, venue_house_number, venue_postal_code, venue_city,
    venue_country, venue_state, venue_point, venue_link,
    space_name, space_total_capacity, space_seating_capacity, space_type,
    space_building_level, space_link, space_accessibility_summary,
    space_accessibility_flags, space_description,
    created_at, modified_at
)
SELECT DISTINCT ON (e.uuid)
    e.uuid,
    e.org_uuid,
    e.venue_uuid,
    e.space_uuid,
    e.release_status,
    e.title,
    e.subtitle,
    e.description,
    e.summary,
    main_image.pluto_image_uuid AS image_uuid,
    e.languages,
    e.tags,
    e.categories,
    COALESCE(
        (SELECT jsonb_agg(jsonb_build_array(type_id, genre_id))
         FROM %[1]s.event_type_link etl WHERE etl.event_uuid = e.uuid),
        '[]'::jsonb
    ),
    e.source_link,
    e.online_link,
    e.occasion_type_id,
    e.max_attendees,
    e.min_age,
    e.max_age,
    e.participation_info,
    e.meeting_point,
    e.ticket_flags,
    e.price_type,
    e.currency,
    e.min_price,
    e.max_price,
    e.visitor_info_flags,
    e.external_id,
    e.custom,
    e.style,
    e.search_text,
    o.name,
    o.contact_email,
    o.contact_phone,
    o.web_link,
    v.name AS venue_name,
    v.street AS venue_street,
    v.house_number AS venue_house_number,
    v.postal_code AS venue_postal_code,
    v.city AS venue_city,
    v.country AS venue_country,
    v.state AS venue_state,
    v.point AS venue_point,
    v.web_link AS venue_web_link,
    s.name,
    s.total_capacity,
    s.seating_capacity,
    s.space_type,
    s.building_level,
    s.web_link,
    s.accessibility_summary,
    s.accessibility_flags,
    s.description,
    NOW(),
    NOW()
FROM %[1]s.event e
LEFT JOIN %[1]s.organization o ON o.uuid = e.org_uuid
LEFT JOIN %[1]s.venue v ON v.uuid = e.venue_uuid
LEFT JOIN %[1]s.space s ON s.uuid = e.space_uuid
JOIN %[1]s.event_date ed ON ed.event_uuid = e.uuid

-- fetch main image
LEFT JOIN LATERAL (
    SELECT pil.pluto_image_uuid
    FROM %[1]s.pluto_image_link pil
    WHERE pil.context = 'event'
      AND pil.context_uuid = e.uuid
      AND pil.identifier = 'main'
    LIMIT 1
) main_image ON TRUE

WHERE e.uuid = ANY($1::uuid[])
  AND ed.start_date >= CURRENT_DATE
ON CONFLICT (event_uuid) DO UPDATE SET
    org_uuid = EXCLUDED.org_uuid,
    venue_uuid = EXCLUDED.venue_uuid,
    space_uuid = EXCLUDED.space_uuid,
    release_status = EXCLUDED.release_status,
    title = EXCLUDED.title,
    subtitle = EXCLUDED.subtitle,
    description = EXCLUDED.description,
    summary = EXCLUDED.summary,
    image_uuid = EXCLUDED.image_uuid,
    languages = EXCLUDED.languages,
    tags = EXCLUDED.tags,
    categories = EXCLUDED.categories,
    types = EXCLUDED.types,
    source_link = EXCLUDED.source_link,
    online_link = EXCLUDED.online_link,
    occasion_type_id = EXCLUDED.occasion_type_id,
    max_attendees = EXCLUDED.max_attendees,
    min_age = EXCLUDED.min_age,
    max_age = EXCLUDED.max_age,
    participation_info = EXCLUDED.participation_info,
    meeting_point = EXCLUDED.meeting_point,
    ticket_flags = EXCLUDED.ticket_flags,
    price_type = EXCLUDED.price_type,
    currency = EXCLUDED.currency,
    min_price = EXCLUDED.min_price,
    max_price = EXCLUDED.max_price,
    visitor_info_flags = EXCLUDED.visitor_info_flags,
    external_id = EXCLUDED.external_id,
    custom = EXCLUDED.custom,
    style = EXCLUDED.style,
    search_text = EXCLUDED.search_text,
    org_name = EXCLUDED.org_name,
    org_contact_email = EXCLUDED.org_contact_email,
    org_contact_phone = EXCLUDED.org_contact_phone,
    org_link = EXCLUDED.org_link,
    venue_name = EXCLUDED.venue_name,
    venue_street = EXCLUDED.venue_street,
    venue_house_number = EXCLUDED.venue_house_number,
    venue_postal_code = EXCLUDED.venue_postal_code,
    venue_city = EXCLUDED.venue_city,
    venue_country = EXCLUDED.venue_country,
    venue_state = EXCLUDED.venue_state,
    venue_point = EXCLUDED.venue_point,
    venue_link = EXCLUDED.venue_link,
    space_name = EXCLUDED.space_name,
    space_total_capacity = EXCLUDED.space_total_capacity,
    space_seating_capacity = EXCLUDED.space_seating_capacity,
    space_type = EXCLUDED.space_type,
    space_building_level = EXCLUDED.space_building_level,
    space_link = EXCLUDED.space_link,
    space_accessibility_summary = EXCLUDED.space_accessibility_summary,
    space_accessibility_flags = EXCLUDED.space_accessibility_flags,
    space_description = EXCLUDED.space_description,
    modified_at = NOW();
`, app.UranusInstance.Config.DbSchema)

		eventDateProjectionUpsertSql = fmt.Sprintf(`
INSERT INTO %[1]s.event_date_projection (
    event_date_uuid, event_uuid, venue_uuid, space_uuid,
    venue_name, venue_street, venue_house_number,
    venue_postal_code, venue_city, venue_country,
    venue_state, venue_point, venue_link,
    space_name, space_total_capacity, space_seating_capacity,
    space_type, space_building_level, space_link,
    space_accessibility_summary, space_accessibility_flags,
    space_description,
    start_date, start_time,
    end_date, end_time,
    entry_time, duration, all_day,
    ticket_link, availability_status_id,
    accessibility_info, custom, created_at, modified_at
)
SELECT DISTINCT ON (ed.uuid)
    ed.uuid,
    ed.event_uuid,
    ed.venue_uuid,
    ed.space_uuid,
    v.name,
    v.street,
    v.house_number,
    v.postal_code,
    v.city,
    v.country,
    v.state,
    v.point,
    v.web_link,
    s.name,
    s.total_capacity,
    s.seating_capacity,
    s.space_type,
    s.building_level,
    s.web_link,
    s.accessibility_summary,
    s.accessibility_flags,
    s.description,
    ed.start_date,
    ed.start_time,
    ed.end_date,
    ed.end_time,
    ed.entry_time,
    ed.duration,
    ed.all_day,
    ed.ticket_link,
    ed.availability_status_id,
    ed.accessibility_info,
    ed.custom,
    NOW(),
    NOW()
FROM %[1]s.event_date ed
LEFT JOIN %[1]s.venue v ON v.uuid = ed.venue_uuid
LEFT JOIN %[1]s.space s ON s.uuid = ed.space_uuid
WHERE ed.uuid = ANY($1::uuid[])
ON CONFLICT (event_date_uuid) DO UPDATE SET
    venue_uuid = EXCLUDED.venue_uuid,
    space_uuid = EXCLUDED.space_uuid,
    venue_name = EXCLUDED.venue_name,
    venue_street = EXCLUDED.venue_street,
    venue_house_number = EXCLUDED.venue_house_number,
    venue_postal_code = EXCLUDED.venue_postal_code,
    venue_city = EXCLUDED.venue_city,
    venue_country = EXCLUDED.venue_country,
    venue_state = EXCLUDED.venue_state,
    venue_point = EXCLUDED.venue_point,
    venue_link = EXCLUDED.venue_link,
    space_name = EXCLUDED.space_name,
    space_total_capacity = EXCLUDED.space_total_capacity,
    space_seating_capacity = EXCLUDED.space_seating_capacity,
    space_type = EXCLUDED.space_type,
    space_building_level = EXCLUDED.space_building_level,
    space_link = EXCLUDED.space_link,
    space_accessibility_summary = EXCLUDED.space_accessibility_summary,
    space_accessibility_flags = EXCLUDED.space_accessibility_flags,
    space_description = EXCLUDED.space_description,
    start_date = EXCLUDED.start_date,
    start_time = EXCLUDED.start_time,
    end_date = EXCLUDED.end_date,
    end_time = EXCLUDED.end_time,
    entry_time = EXCLUDED.entry_time,
    duration = EXCLUDED.duration,
    all_day = EXCLUDED.all_day,
    ticket_link = EXCLUDED.ticket_link,
    availability_status_id = EXCLUDED.availability_status_id,
    accessibility_info = EXCLUDED.accessibility_info,
    custom = EXCLUDED.custom,
    modified_at = NOW();
`, app.UranusInstance.Config.DbSchema)
	})
}

func fetchUuids(
	ctx context.Context,
	tx pgx.Tx,
	query string,
	uuids []string,
) ([]string, error) {

	fmt.Println(query)
	rows, err := tx.Query(ctx, query, uuids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []string
	for rows.Next() {
		var uuid string
		if err := rows.Scan(&uuid); err != nil {
			return nil, err
		}
		result = append(result, uuid)
	}
	return result, nil
}

func uniqueInts(ids []int) []int {
	seen := make(map[int]struct{}, len(ids))
	out := make([]int, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	return out
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
