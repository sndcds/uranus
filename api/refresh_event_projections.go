package api

import (
	"context"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/sndcds/uranus/app"
)

type affectedQueries struct {
	EventIds     string
	EventDateIds string
}

var (
	queryOnce sync.Once
	queries   map[string]affectedQueries
)

var (
	initProjectionQueries        sync.Once
	eventProjectionUpsertSQL     string
	eventDateProjectionUpsertSQL string
)

func RefreshEventProjections(
	ctx context.Context,
	tx pgx.Tx,
	sourceTable string,
	ids []int,
) error {

	if len(ids) == 0 {
		return nil
	}

	initQueries()

	ids = uniqueInts(ids)
	fmt.Println("RefreshEventProjections sourceTable:", sourceTable, "ids:", ids)

	q, ok := queries[sourceTable]
	if !ok {
		return fmt.Errorf("unsupported source table: %s", sourceTable)
	}

	// --- Refresh events ---
	if q.EventIds != "" {
		eventIds, err := fetchIds(ctx, tx, q.EventIds, ids)
		if err != nil {
			return err
		}
		fmt.Println("RefreshEventProjections eventIds:", eventIds)
		if len(eventIds) > 0 {
			if err := upsertEventProjection(ctx, tx, eventIds); err != nil {
				return err
			}
		}
	}

	// --- Refresh event dates ---
	if q.EventDateIds != "" {
		eventDateIds, err := fetchIds(ctx, tx, q.EventDateIds, ids)
		if err != nil {
			return err
		}
		fmt.Println("RefreshEventProjections eventDateIds:", eventDateIds)
		if len(eventDateIds) > 0 {
			if err := upsertEventDateProjection(ctx, tx, eventDateIds); err != nil {
				return err
			}
		}
	}

	return nil
}

func upsertEventProjection(ctx context.Context, tx pgx.Tx, eventIDs []int) error {
	if len(eventIDs) == 0 {
		return nil
	}

	initProjectionSQL()

	_, err := tx.Exec(ctx, eventProjectionUpsertSQL, eventIDs)
	return err
}

func upsertEventDateProjection(ctx context.Context, tx pgx.Tx, eventDateIDs []int) error {
	if len(eventDateIDs) == 0 {
		return nil
	}

	initProjectionSQL()

	// fmt.Println("UpsertEventProjection eventDateProjectionUpsertSQL:", eventDateProjectionUpsertSQL)

	_, err := tx.Exec(ctx, eventDateProjectionUpsertSQL, eventDateIDs)
	return err
}

func initQueries() {
	queryOnce.Do(func() {
		schema := app.UranusInstance.Config.DbSchema
		queries = map[string]affectedQueries{
			"organization": {
				EventIds: fmt.Sprintf(`
					SELECT id
					FROM %s.event
					WHERE organization_id = ANY($1)
				`, schema),

				EventDateIds: fmt.Sprintf(`
					SELECT ed.id
					FROM %s.event_date ed
					JOIN %s.event e ON e.id = ed.event_id
					WHERE e.organization_id = ANY($1)
				`, schema, schema),
			},

			"venue": {
				EventIds: fmt.Sprintf(`
					SELECT id
					FROM %s.event
					WHERE venue_id = ANY($1)
				`, schema),

				EventDateIds: fmt.Sprintf(`
					SELECT id
					FROM %s.event_date
					WHERE venue_id = ANY($1)
				`, schema),
			},

			"space": {
				EventIds: fmt.Sprintf(`
					SELECT id
					FROM %s.event
					WHERE space_id = ANY($1)
				`, schema),

				EventDateIds: fmt.Sprintf(`
					SELECT id
					FROM %s.event_date
					WHERE space_id = ANY($1)
				`, schema),
			},

			"pluto_image": {
				EventIds: fmt.Sprintf(`
					SELECT id
					FROM %s.event
					WHERE image1_id = ANY($1)
				`, schema),

				EventDateIds: "", // image does not affect event_date
			},

			"event": {
				EventIds: fmt.Sprintf(`
					SELECT id
					FROM %s.event
					WHERE id = ANY($1)
				`, schema),

				EventDateIds: fmt.Sprintf(`
					SELECT id
					FROM %s.event_date
					WHERE event_id = ANY($1)
				`, schema),
			},

			"event_date": {
				EventIds: "", // event_date update only affects itself, not the parent event
				EventDateIds: fmt.Sprintf(`
					SELECT id
					FROM %s.event_date
					WHERE id = ANY($1)
				`, schema),
			},
		}
	})
}

func initProjectionSQL() {
	initProjectionQueries.Do(func() {
		eventProjectionUpsertSQL = fmt.Sprintf(`
INSERT INTO %[1]s.event_projection (
    event_id, organization_id, venue_id, space_id, location_id, release_status_id,
    title, subtitle, description, summary, image_id, languages, tags, types,
    source_url, online_event_url, occasion_type_id, max_attendees, min_age, max_age,
    participation_info, meeting_point, ticket_advance, ticket_required,
    registration_required, price_type_id, currency_code, min_price, max_price,
    external_id, custom, style, search_text,
    organization_name, organization_contact_email, organization_contact_phone, organization_website_url,
	venue_name, venue_street, venue_house_number, venue_postal_code, venue_city,
	venue_country_code, venue_state_code, venue_geo_pos, venue_website_url,
    space_name, space_total_capacity, space_seating_capacity, space_type_id,
    space_building_level, space_website_url, space_accessibility_summary,
    space_accessibility_flags, space_description,
    created_at, modified_at
)
SELECT DISTINCT ON (e.id)
    e.id, e.organization_id, e.venue_id, e.space_id, e.location_id, e.release_status_id,
    e.title, e.subtitle, e.description, e.summary, e.image1_id, e.languages, e.tags,
    COALESCE(
        (SELECT jsonb_agg(jsonb_build_array(type_id, genre_id))
         FROM %[1]s.event_type_link etl WHERE etl.event_id = e.id),
        '[]'::jsonb
    ),
    e.source_url, e.online_event_url, e.occasion_type_id, e.max_attendees,
    e.min_age, e.max_age, e.participation_info, e.meeting_point,
    e.ticket_advance, e.ticket_required, e.registration_required,
    e.price_type_id, e.currency_code, e.min_price, e.max_price,
    e.external_id, e.custom, e.style, e.search_text,
    o.name, o.contact_email, o.contact_phone, o.website_url,
	COALESCE(v.name, el.name) AS venue_name,
    COALESCE(v.street, el.street) AS venue_street,
    COALESCE(v.house_number, el.house_number) AS venue_house_number,
    COALESCE(v.postal_code, el.postal_code) AS venue_postal_code,
    COALESCE(v.city, el.city) AS venue_city,
    COALESCE(v.country_code, el.country_code) AS venue_country_code,
    COALESCE(v.state_code, el.state_code) AS venue_state_code,
    COALESCE(v.wkb_pos, el.wkb_pos) AS venue_geo_pos,
    v.website_url AS venue_website_url,
    s.name, s.total_capacity, s.seating_capacity, s.space_type_id,
    s.building_level, s.website_url, s.accessibility_summary,
    s.accessibility_flags, s.description,
    NOW(), NOW()
FROM %[1]s.event e
LEFT JOIN %[1]s.organization o ON o.id = e.organization_id
LEFT JOIN %[1]s.venue v ON v.id = e.venue_id
LEFT JOIN %[1]s.space s ON s.id = e.space_id
LEFT JOIN %[1]s.event_location el ON el.id = e.location_id
JOIN %[1]s.event_date ed ON ed.event_id = e.id
WHERE e.id = ANY($1)
AND ed.start_date >= CURRENT_DATE
ON CONFLICT (event_id) DO UPDATE SET
    organization_id = EXCLUDED.organization_id,
    venue_id = EXCLUDED.venue_id,
    space_id = EXCLUDED.space_id,
    location_id = EXCLUDED.location_id,
    release_status_id = EXCLUDED.release_status_id,
    title = EXCLUDED.title,
    subtitle = EXCLUDED.subtitle,
    description = EXCLUDED.description,
    summary = EXCLUDED.summary,
    image_id = EXCLUDED.image_id,
    languages = EXCLUDED.languages,
    tags = EXCLUDED.tags,
    types = EXCLUDED.types,
    source_url = EXCLUDED.source_url,
    online_event_url = EXCLUDED.online_event_url,
    occasion_type_id = EXCLUDED.occasion_type_id,
    max_attendees = EXCLUDED.max_attendees,
    min_age = EXCLUDED.min_age,
    max_age = EXCLUDED.max_age,
    participation_info = EXCLUDED.participation_info,
    meeting_point = EXCLUDED.meeting_point,
    ticket_advance = EXCLUDED.ticket_advance,
    ticket_required = EXCLUDED.ticket_required,
    registration_required = EXCLUDED.registration_required,
    price_type_id = EXCLUDED.price_type_id,
    currency_code = EXCLUDED.currency_code,
    min_price = EXCLUDED.min_price,
    max_price = EXCLUDED.max_price,
    external_id = EXCLUDED.external_id,
    custom = EXCLUDED.custom,
    style = EXCLUDED.style,
    search_text = EXCLUDED.search_text,
    organization_name = EXCLUDED.organization_name,
    organization_contact_email = EXCLUDED.organization_contact_email,
    organization_contact_phone = EXCLUDED.organization_contact_phone,
    organization_website_url = EXCLUDED.organization_website_url,
    venue_name = EXCLUDED.venue_name,
    venue_street = EXCLUDED.venue_street,
    venue_house_number = EXCLUDED.venue_house_number,
    venue_postal_code = EXCLUDED.venue_postal_code,
    venue_city = EXCLUDED.venue_city,
    venue_country_code = EXCLUDED.venue_country_code,
    venue_state_code = EXCLUDED.venue_state_code,
    venue_geo_pos = EXCLUDED.venue_geo_pos,
    venue_website_url = EXCLUDED.venue_website_url,
    space_name = EXCLUDED.space_name,
    space_total_capacity = EXCLUDED.space_total_capacity,
    space_seating_capacity = EXCLUDED.space_seating_capacity,
    space_type_id = EXCLUDED.space_type_id,
    space_building_level = EXCLUDED.space_building_level,
    space_website_url = EXCLUDED.space_website_url,
    space_accessibility_summary = EXCLUDED.space_accessibility_summary,
    space_accessibility_flags = EXCLUDED.space_accessibility_flags,
    space_description = EXCLUDED.space_description,
    modified_at = NOW();
`, app.UranusInstance.Config.DbSchema)

		eventDateProjectionUpsertSQL = fmt.Sprintf(`
INSERT INTO %[1]s.event_date_projection (
    event_date_id, event_id, venue_id, space_id,
    venue_name, venue_street, venue_house_number,
    venue_postal_code, venue_city, venue_country_code,
    venue_state_code, venue_geo_pos, venue_website_url,
    space_name, space_total_capacity, space_seating_capacity,
    space_type_id, space_building_level, space_website_url,
    space_accessibility_summary, space_accessibility_flags,
    space_description,
    start_date, start_time,
    end_date, end_time,
    entry_time, duration, all_day,
    visitor_info_flags, ticket_link, availability_status_id,
    accessibility_info, custom, created_at, modified_at
)
SELECT DISTINCT ON (ed.id)
    ed.id,
    ed.event_id,
    ed.venue_id,
    ed.space_id,
    v.name,
    v.street,
    v.house_number,
    v.postal_code,
    v.city,
    v.country_code,
    v.state_code,
    v.wkb_pos,
    v.website_url,
    s.name,
    s.total_capacity,
    s.seating_capacity,
    s.space_type_id,
    s.building_level,
    s.website_url,
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
    ed.visitor_info_flags,
    ed.ticket_link,
    ed.availability_status_id,
    ed.accessibility_info,
    ed.custom,
    NOW(),
    NOW()
FROM %[1]s.event_date ed
LEFT JOIN %[1]s.venue v ON v.id = ed.venue_id
LEFT JOIN %[1]s.space s ON s.id = ed.space_id
WHERE ed.id = ANY($1)
AND ed.start_date >= CURRENT_DATE
ON CONFLICT (event_date_id) DO UPDATE SET
    venue_id = EXCLUDED.venue_id,
    space_id = EXCLUDED.space_id,
    venue_name = EXCLUDED.venue_name,
    venue_street = EXCLUDED.venue_street,
    venue_house_number = EXCLUDED.venue_house_number,
    venue_postal_code = EXCLUDED.venue_postal_code,
    venue_city = EXCLUDED.venue_city,
    venue_country_code = EXCLUDED.venue_country_code,
    venue_state_code = EXCLUDED.venue_state_code,
    venue_geo_pos = EXCLUDED.venue_geo_pos,
    venue_website_url = EXCLUDED.venue_website_url,
    space_name = EXCLUDED.space_name,
    space_total_capacity = EXCLUDED.space_total_capacity,
    space_seating_capacity = EXCLUDED.space_seating_capacity,
    space_type_id = EXCLUDED.space_type_id,
    space_building_level = EXCLUDED.space_building_level,
    space_website_url = EXCLUDED.space_website_url,
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
    visitor_info_flags = EXCLUDED.visitor_info_flags,
    ticket_link = EXCLUDED.ticket_link,
    availability_status_id = EXCLUDED.availability_status_id,
    accessibility_info = EXCLUDED.accessibility_info,
    custom = EXCLUDED.custom,
    modified_at = NOW();
`, app.UranusInstance.Config.DbSchema)
	})
}

func fetchIds(
	ctx context.Context,
	tx pgx.Tx,
	query string,
	ids []int,
) ([]int, error) {

	rows, err := tx.Query(ctx, query, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		result = append(result, id)
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
