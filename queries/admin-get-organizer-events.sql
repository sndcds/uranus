WITH event_data AS (
    SELECT
        ed.id AS event_date_id,
        ed.event_id,
        ed.venue_id,
        ed.space_id,
        ed.start_date,
        ed.start_time,
        ed.end_date,
        ed.end_time,
        ed.entry_time,
        ed.duration,
        ed.accessibility_info,
        ed.visitor_info_flags
    FROM {{schema}}.event_date ed
)
SELECT
    e.id AS event_id,
    event_date_id,
    e.title AS event_title,
    e.subtitle AS event_subtitle,
    e.organizer_id AS event_organizer_id,
    eo.name AS event_organizer_name,
    TO_CHAR(ed.start_date, 'YYYY-MM-DD') AS start_date,
    TO_CHAR(ed.start_time, 'HH24:MI') AS start_time,
    TO_CHAR(ed.end_date, 'YYYY-MM-DD') AS end_date,
    TO_CHAR(ed.end_time, 'HH24:MI') AS end_time,
    e.release_status_id,
    est.name AS release_status_name,
    TO_CHAR(e.release_date, 'YYYY-MM-DD') AS release_date,
    COALESCE(ed.venue_id, e.venue_id) AS venue_id,
    v.name AS venue_name,

    -- Space: only from event_date if event_date.venue_id exists, else null
    CASE WHEN ed.venue_id IS NOT NULL THEN s.id ELSE NULL END AS space_id,
    CASE WHEN ed.venue_id IS NOT NULL THEN s.name ELSE NULL END AS space_name,

    ST_X(v.wkb_geometry) AS venue_lon,
    ST_Y(v.wkb_geometry) AS venue_lat,
    e.image1_id AS image_id,
    et_data.event_types,

    -- Permissions via bitmask
    COALESCE((uel.permissions & (1<<25)) <> 0, FALSE)
        OR COALESCE((uol.permissions & (1<<25)) <> 0, FALSE)
        OR COALESCE((uvl.permissions & (1<<25)) <> 0, FALSE) AS can_edit_event,
    COALESCE((uel.permissions & (1<<26)) <> 0, FALSE)
        OR COALESCE((uol.permissions & (1<<26)) <> 0, FALSE)
        OR COALESCE((uvl.permissions & (1<<26)) <> 0, FALSE) AS can_delete_event,
    COALESCE((uel.permissions & (1<<27)) <> 0, FALSE)
        OR COALESCE((uol.permissions & (1<<27)) <> 0, FALSE)
        OR COALESCE((uvl.permissions & (1<<27)) <> 0, FALSE) AS can_release_event,

    ROW_NUMBER() OVER (
            PARTITION BY ed.event_id
            ORDER BY ed.start_date, ed.start_time
        ) AS time_series_index,
    COUNT(ed.event_date_id) OVER (PARTITION BY ed.event_id) AS time_series

FROM event_data ed
LEFT JOIN {{schema}}.event e ON ed.event_id = e.id
LEFT JOIN {{schema}}.venue v ON COALESCE(ed.venue_id, e.venue_id) = v.id
LEFT JOIN {{schema}}.space s ON ed.space_id = s.id
LEFT JOIN {{schema}}.organizer o ON v.organizer_id = o.id
LEFT JOIN {{schema}}.organizer eo ON e.organizer_id = eo.id
LEFT JOIN {{schema}}.event_location el ON e.location_id = el.id

LEFT JOIN {{schema}}.event_status est ON est.status_id = e.release_status_id AND est.iso_639_1 = $3

LEFT JOIN LATERAL (
    SELECT jsonb_agg(
        event_type ORDER BY event_type.type_name, event_type.genre_name
    ) AS event_types
    FROM (
        SELECT DISTINCT
            etl.type_id,
            et.name AS type_name,
            COALESCE(gt.type_id, 0) AS genre_id,
            gt.name AS genre_name
        FROM {{schema}}.event_type_link etl
        JOIN {{schema}}.event_type et ON et.type_id = etl.type_id AND et.iso_639_1 = $3
        LEFT JOIN {{schema}}.genre_type gt ON gt.type_id = etl.genre_id AND gt.iso_639_1 = $3
        WHERE etl.event_id = e.id
    ) event_type
) et_data ON true

-- User links with permissions bitmask
LEFT JOIN {{schema}}.user_event_link uel ON uel.event_id = e.id AND uel.user_id = $4
LEFT JOIN {{schema}}.user_organizer_link uol ON uol.organizer_id = e.organizer_id AND uol.user_id = $4
LEFT JOIN {{schema}}.user_venue_link uvl ON uvl.venue_id = e.venue_id AND uvl.user_id = $4

WHERE eo.id = $1
AND (ed.start_date + COALESCE(ed.start_time, '00:00:00'::time)) >= $2::timestamp
ORDER BY ed.start_date, ed.start_time