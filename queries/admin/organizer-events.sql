WITH event_data AS (
    SELECT
        ed.id AS event_date_id,
        ed.event_id,
        ed.space_id,
        ed.start,
        ed.end,
        ed.entry_time,
        ed.duration,
        ed.accessibility_flags,
        ed.visitor_info_flags
    FROM {{schema}}.event_date ed
    )
SELECT
    e.id AS event_id,
    e.title AS event_title,
    e.subtitle AS event_subtitle,
    e.organizer_id AS event_organizer_id,
    eo.name AS event_organizer_name,
    TO_CHAR(ed.start, 'YYYY-MM-DD') AS start_date,
    TO_CHAR(ed.start, 'HH24:MI') AS start_time,
    TO_CHAR(ed.end, 'YYYY-MM-DD') AS end_date,
    TO_CHAR(ed.end, 'HH24:MI') AS end_time,
    v.id AS venue_id,
    v.name AS venue_name,
    COALESCE(s.id, es.id) AS space_id,
    COALESCE(s.name, es.name) AS space_name,
    ST_X(v.wkb_geometry) AS venue_lon,
    ST_Y(v.wkb_geometry) AS venue_lat,
    eil.pluto_image_id AS image_id,
    et_data.event_types

FROM event_data ed
    LEFT JOIN {{schema}}.event e ON ed.event_id = e.id
    LEFT JOIN {{schema}}.space s ON ed.space_id = s.id
    LEFT JOIN {{schema}}.space es ON e.space_id = es.id
    JOIN {{schema}}.venue v ON e.venue_id = v.id
    LEFT JOIN {{schema}}.organizer o ON v.organizer_id = o.id
    LEFT JOIN {{schema}}.organizer eo ON e.organizer_id = eo.id
    LEFT JOIN {{schema}}.event_image_links eil ON e.id = eil.event_id AND eil.main_image = TRUE

    LEFT JOIN LATERAL (
        SELECT jsonb_agg(DISTINCT jsonb_build_object(
            'type_id', etl.type_id,
            'type_name', et.name,
            'genre_id', COALESCE(gt.type_id, 0),
            'genre_name', gt.name
        )) AS event_types
        FROM {{schema}}.event_type_links etl
        JOIN {{schema}}.event_type et ON et.type_id = etl.type_id AND et.iso_639_1 = $3
        LEFT JOIN {{schema}}.genre_type gt ON gt.type_id = etl.genre_id AND gt.iso_639_1 = $3
        WHERE etl.event_id = e.id
    ) et_data ON true


WHERE (o.id = $1 OR o.id IS NULL)   -- keep event_dates even if venue.organizer is missing
  AND ed.start::date >= $2
ORDER BY ed.start