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
    {{event-date-conditions}}
    )
SELECT
    {{mode-dependent-select}}

FROM event_data ed
    JOIN {{schema}}.event e ON ed.event_id = e.id AND e.release_status_id >= 3
    JOIN {{schema}}.organizer o ON e.organizer_id = o.id
    LEFT JOIN {{schema}}.space s ON ed.space_id = s.id
    LEFT JOIN {{schema}}.space es ON e.space_id = es.id
    LEFT JOIN {{schema}}.venue v ON v.id = e.venue_id

    LEFT JOIN LATERAL (
    SELECT
    TRUE AS has_main_image,
    eil.pluto_image_id AS id,
    0 AS focus_x,
    0 AS focus_y
    FROM {{schema}}.event_image_links eil
    WHERE eil.event_id = e.id AND eil.main_image = TRUE
    LIMIT 1
    ) img_data ON true

    LEFT JOIN LATERAL (
    SELECT jsonb_agg(DISTINCT jsonb_build_object(
    'type_id', etl.type_id,
    'type_name', et.name,
    'genre_id', COALESCE(gt.type_id, 0),
    'genre_name', gt.name
    )) AS event_types
    FROM {{schema}}.event_type_links etl
    JOIN {{schema}}.event_type et
    ON et.type_id = etl.type_id
    AND et.iso_639_1 = $1
    LEFT JOIN {{schema}}.genre_type gt
    ON gt.type_id = etl.genre_id
    AND gt.iso_639_1 = $1
    WHERE etl.event_id = e.id
    ) et_data ON true

    LEFT JOIN LATERAL (
    SELECT jsonb_agg(name) AS accessibility_flag_names
    FROM {{schema}}.accessibility_flags f
    WHERE (ed.accessibility_flags & (1::BIGINT << f.flag)) = (1::BIGINT << f.flag) AND f.iso_639_1 = $1
    ) acc_flags ON true

    LEFT JOIN LATERAL (
    SELECT jsonb_agg(name) AS visitor_info_flag_names
    FROM {{schema}}.visitor_information_flags f
    WHERE (ed.visitor_info_flags & (1::BIGINT << f.flag)) = (1::BIGINT << f.flag) AND f.iso_639_1 = $1
    ) vis_flags ON true

    {{conditions}}
    {{order}}
    {{limit}}