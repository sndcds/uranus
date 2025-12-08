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
        ed.visitor_info_flags
    FROM {{schema}}.event_date ed
    {{event-date-conditions}}
    )
SELECT
    {{mode-dependent-select}}

FROM event_data ed

-- Base event & organizer
JOIN {{schema}}.event e
ON ed.event_id = e.id
AND e.release_status_id >= 3

JOIN {{schema}}.organizer o
ON e.organizer_id = o.id

-- Space overrides
LEFT JOIN {{schema}}.space s
ON ed.space_id = s.id            -- event_date space override

LEFT JOIN {{schema}}.space es
ON e.space_id = es.id            -- event-level space

-- Venue overrides
LEFT JOIN {{schema}}.venue v_ev
ON v_ev.id = e.venue_id          -- event-level venue

LEFT JOIN {{schema}}.venue v_ed
ON v_ed.id = ed.venue_id         -- event_date-specific venue override

-- Main image
LEFT JOIN LATERAL (
    SELECT
        pli.id AS id,
        pli.focus_x AS focus_x,
        pli.focus_y AS focus_y
    FROM {{schema}}.pluto_image pli
    WHERE pli.id = e.image1_id
    LIMIT 1
) image_data ON true

-- Types and Genres
LEFT JOIN LATERAL (
    SELECT jsonb_agg(
        DISTINCT jsonb_build_object(
            'type_id', etl.type_id,
            'type_name', et.name,
            'genre_id', COALESCE(gt.type_id, 0),
            'genre_name', gt.name
        )
    ) AS event_types
    FROM {{schema}}.event_type_link etl
    JOIN {{schema}}.event_type et
    ON et.type_id = etl.type_id
    AND et.iso_639_1 = $1
    LEFT JOIN {{schema}}.genre_type gt
    ON gt.type_id = etl.genre_id
    AND gt.iso_639_1 = $1
    WHERE etl.event_id = e.id
) et_data ON true

-- Visitor info flags
LEFT JOIN LATERAL (
    SELECT jsonb_agg(name) AS visitor_info_flag_names
    FROM {{schema}}.visitor_information_flags f
    WHERE (ed.visitor_info_flags & (1::BIGINT << f.flag)) = (1::BIGINT << f.flag)
    AND f.iso_639_1 = $1
) vis_flags ON true

{{conditions}}
{{order}}
{{limit}}