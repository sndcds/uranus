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
    JOIN {{schema}}.event e ON e.id = ed.event_id
    {{event-date-conditions}}
),
venue_data AS (
    SELECT
        ed.event_date_id AS venue_event_date,
        COALESCE(v_ed.id, v_ev.id) AS venue_id,
        COALESCE(v_ed.name, v_ev.name) AS venue_name,
        COALESCE(v_ed.street, v_ev.street) AS venue_street,
        COALESCE(v_ed.house_number, v_ev.house_number) AS venue_house_number,
        COALESCE(v_ed.postal_code, v_ev.postal_code) AS venue_postal_code,
        COALESCE(v_ed.city, v_ev.city) AS venue_city,
        COALESCE(v_ed.country_code, v_ev.country_code) AS venue_country_code,
        COALESCE(v_ed.state_code, v_ev.state_code) AS venue_state_code,
        COALESCE(ST_X(v_ed.wkb_pos), ST_X(v_ev.wkb_pos)) AS venue_lon,
        COALESCE(ST_Y(v_ed.wkb_pos), ST_Y(v_ev.wkb_pos)) AS venue_lat,
        COALESCE(v_ed.wkb_pos, v_ev.wkb_pos) AS venue_wkb_pos
    FROM event_data ed
    LEFT JOIN {{schema}}.venue v_ev ON v_ev.id = (SELECT e.venue_id FROM {{schema}}.event e WHERE e.id = ed.event_id)
    LEFT JOIN {{schema}}.venue v_ed ON v_ed.id = ed.venue_id
)

SELECT
    {{mode-dependent-select}}

FROM event_data ed

-- Base event, organization, venue
JOIN {{schema}}.event e
ON ed.event_id = e.id
AND e.release_status_id >= 3

JOIN {{schema}}.organization o
ON e.organization_id = o.id

JOIN venue_data vd
ON vd.venue_event_date = ed.event_date_id

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