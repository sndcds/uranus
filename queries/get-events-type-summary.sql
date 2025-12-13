WITH event_date AS (
    SELECT
        ed.id AS event_date_id,
        ed.event_id,
        e.organizer_id,
        o.name AS organizer_name,
        COALESCE(ed.venue_id, e.venue_id) AS venue_id,
        COALESCE(v_ed.name, v_e.name) AS venue_name,
        COALESCE(v_ed.city, v_e.city) AS venue_city
    FROM {{schema}}.event_date ed
    JOIN {{schema}}.event e ON ed.event_id = e.id AND e.release_status_id >= 3
    JOIN {{schema}}.organizer o ON e.organizer_id = o.id
    LEFT JOIN {{schema}}.venue v_e ON e.venue_id = v_e.id
    LEFT JOIN {{schema}}.venue v_ed ON ed.venue_id = v_ed.id
    {{event-date-conditions}}
),
event_type_data AS (
    -- Deduplicate type per event per event_date
    SELECT DISTINCT
        ed.event_date_id,
        etd.type_id,
        et.name AS type_name
    FROM event_date ed
    JOIN {{schema}}.event_type_link etd ON etd.event_id = ed.event_id
    JOIN {{schema}}.event_type et ON et.type_id = etd.type_id AND et.iso_639_1 = $1
)
SELECT json_build_object(
    'organizer_summary', (
        SELECT json_agg(organizer_summary)
        FROM (
            SELECT
                ed.organizer_id AS id,
                ed.organizer_name AS name,
                COUNT(ed.event_date_id) AS event_date_count
            FROM event_date ed
            GROUP BY ed.organizer_id, ed.organizer_name
            ORDER BY name ASC
        ) organizer_summary
    ),
    'total', (SELECT COUNT(*) FROM event_date),
    'type_summary', (
        SELECT json_agg(type_summary)
        FROM (
            SELECT
                COUNT(etd.event_date_id) AS count,
                etd.type_id,
                etd.type_name
            FROM event_type_data etd
            GROUP BY etd.type_id, etd.type_name
            ORDER BY count DESC
        ) type_summary
    ),
    'venues_summary', (
        SELECT json_agg(venue_summary)
        FROM (
            SELECT
                ed.venue_id AS id,
                ed.venue_name AS name,
                ed.venue_city AS city,
                COUNT(ed.event_date_id) AS event_date_count
            FROM event_date ed
            WHERE ed.venue_id IS NOT NULL
            GROUP BY ed.venue_id, ed.venue_name, ed.venue_city
            ORDER BY name ASC
        ) venue_summary
    )
) AS summary
{{conditions}}