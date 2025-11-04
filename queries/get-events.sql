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
    e.id AS id,
    e.title AS title,
    e.subtitle AS subtitle,
    e.description AS description,
    e.teaser_text AS teaser_text,
    e.participation_info AS participation_info,
    e.meeting_point AS meeting_point,
    e.languages,

    o.id AS organizer_id,
    o.name AS organizer_name,

    v.id AS venue_id,
    v.name AS venue_name,
    v.street AS venue_street,
    v.house_number AS venue_house_number,
    v.postal_code AS venue_postal_code,
    v.city AS venue_city,
    v.country_code AS venue_country,
    v.state_code AS venue_state,
    ST_AsText(v.wkb_geometry) AS venue_geometry,
    ST_X(v.wkb_geometry) AS venue_lon,
    ST_Y(v.wkb_geometry) AS venue_lat,

    COALESCE(s.id, es.id) AS space_id,
    COALESCE(s.name, es.name) AS space_name,
    COALESCE(s.total_capacity, es.total_capacity) AS space_total_capacity,
    COALESCE(s.seating_capacity, es.seating_capacity) AS space_seating_capacity,
    COALESCE(s.building_level, es.building_level) AS space_building_level,
    COALESCE(s.website_url, es.website_url) AS space_url,

    TO_CHAR(ed.start, 'YYYY-MM-DD') AS start_date,
    TO_CHAR(ed.start, 'HH24:MI') AS start_time,
    TO_CHAR(ed.end, 'YYYY-MM-DD') AS end_date,
    TO_CHAR(ed.end, 'HH24:MI') AS end_time,
    TO_CHAR(ed.entry_time, 'HH24:MI') AS entry_time,
    ed.duration AS duration,

    ed.accessibility_flags AS accessibility_flags,
    ed.visitor_info_flags AS visitor_info_flags,

    acc_flags.accessibility_flag_names AS accessibility_flag_names,
    vis_flags.visitor_info_flag_names AS visitor_info_flag_names,

    (img_data.has_main_image) AS has_main_image,
    img_data.id AS image_id,
    img_data.focus_x AS image_focus_x,
    img_data.focus_y AS image_focus_y,

    et_data.event_types

FROM event_data ed
    JOIN {{schema}}.event e ON ed.event_id = e.id
    JOIN {{schema}}.organizer o ON e.organizer_id = o.id
    LEFT JOIN {{schema}}.space s ON ed.space_id = s.id
    LEFT JOIN {{schema}}.space es ON e.space_id = es.id
    -- LEFT JOIN {{schema}}.venue v ON COALESCE(s.venue_id, es.venue_id) = v.id
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