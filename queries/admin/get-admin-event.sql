WITH event_data AS (
    SELECT
        ed.id AS event_date_id,
        ed.event_id,
        ed.space_id,
        TO_CHAR(ed.start, 'YYYY-MM-DD') AS start_date,
        TO_CHAR(ed.start, 'HH24:MI') AS start_time,
        TO_CHAR(ed.end, 'YYYY-MM-DD') AS end_date,
        TO_CHAR(ed.end, 'HH24:MI') AS end_time,
        TO_CHAR(ed.entry_time, 'HH24:MI') AS entry_time,
        ed.duration,
        ed.accessibility_flags,
        ed.visitor_info_flags
    FROM {{schema}}.event_date ed
WHERE ed.event_id = $1
    )
SELECT
    e.id AS id,
    e.title,
    e.subtitle,
    e.description,
    e.teaser_text,
    e.participation_info,
    e.meeting_point,
    o.id AS organizer_id,
    o.name AS organizer_name,
    v.id AS venue_id,
    v.name AS venue_name,
    v.street,
    v.house_number,
    v.postal_code,
    v.city,
    v.country_code,
    v.state_code,
    ST_AsText(v.wkb_geometry) AS venue_geometry,
    ST_X(v.wkb_geometry) AS venue_lon,
    ST_Y(v.wkb_geometry) AS venue_lat,
    COALESCE(s.id, es.id) AS space_id,
    COALESCE(s.name, es.name) AS space_name,

    -- Aggregate all event dates into a single JSON array
    jsonb_agg(
            jsonb_build_object(
                    'event_date_id', ed.event_date_id,
                    'start_date', ed.start_date,
                    'start_time', ed.start_time,
                    'end_date', ed.end_date,
                    'end_time', ed.end_time,
                    'entry_time', ed.entry_time,
                    'space_id', ed.space_id,
                    'duration', ed.duration,
                    'accessibility_flags', ed.accessibility_flags,
                    'visitor_info_flags', ed.visitor_info_flags
            )
    ) AS event_dates,

    -- Images
    (SELECT jsonb_agg(jsonb_build_object(
            'has_main_image', TRUE,
            'id', eil.pluto_image_id,
            'focus_x', 0,
            'focus_y', 0
                      ))
     FROM {{schema}}.event_image_links eil
WHERE eil.event_id = e.id AND eil.main_image = TRUE
    ) AS images,

-- Event types
    (SELECT jsonb_agg(jsonb_build_object(
    'type_id', etl.type_id,
    'type_name', et.name,
    'genre_id', COALESCE(gt.type_id, 0),
    'genre_name', gt.name
    ))
FROM {{schema}}.event_type_links etl
    JOIN {{schema}}.event_type et ON et.type_id = etl.type_id AND et.iso_639_1 = $2
    LEFT JOIN {{schema}}.genre_type gt ON gt.type_id = etl.genre_id AND gt.iso_639_1 = $2
WHERE etl.event_id = e.id
    ) AS event_types,

-- Event URLs
    (SELECT jsonb_agg(jsonb_build_object(
    'id', eu.id,
    'url_type', eu.url_type,
    'url', eu.url,
    'title', eu.title
    ))
FROM {{schema}}.event_url eu
WHERE eu.event_id = $1
    ) AS event_urls

FROM {{schema}}.event e
    LEFT JOIN event_data ed ON ed.event_id = e.id
    LEFT JOIN {{schema}}.organizer o ON e.organizer_id = o.id
    LEFT JOIN {{schema}}.space s ON s.id = ed.space_id
    LEFT JOIN {{schema}}.space es ON es.id = e.space_id
    LEFT JOIN {{schema}}.venue v ON v.id = e.venue_id

GROUP BY
    e.id, e.title, e.subtitle, e.description, e.teaser_text,
    e.participation_info, e.meeting_point,
    o.id, o.name,
    v.id, v.name, v.street, v.house_number, v.postal_code, v.city, v.country_code, v.state_code,
    ST_AsText(v.wkb_geometry), ST_X(v.wkb_geometry), ST_Y(v.wkb_geometry),
    COALESCE(s.id, es.id), COALESCE(s.name, es.name)

    LIMIT 1;