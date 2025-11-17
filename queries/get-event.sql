SELECT
    e.id AS event_id,
    e.title,
    e.subtitle,
    e.description,
    e.teaser_text,
    e.participation_info,
    e.meeting_point,
    e.languages,

    o.id AS organizer_id,
    o.name AS organizer_name,

    -- v.id AS venue_id,
    -- v.name AS venue_name,
    -- v.street AS venue_street,
    -- v.house_number AS venue_house_number,
    -- v.postal_code AS venue_postal_code,
    -- v.city AS venue_city,
    -- v.country_code AS venue_country,
    -- v.state_code AS venue_state,
    -- ST_X(v.wkb_geometry) AS venue_lon,
    -- ST_Y(v.wkb_geometry) AS venue_lat,

    -- s.id AS space_id,
    -- s.name AS space_name,
    -- s.total_capacity AS space_total_capacity,
    -- s.seating_capacity AS space_seating_capacity,
    -- s.building_level AS space_building_level,
    -- s.website_url AS space_url,

    img_data.has_main_image,
    img_data.id AS image_id,
    pimg.width AS image_width,
    pimg.height AS image_height,
    pimg.mime_type AS image_mime_type,
    pimg.alt_text AS image_alt_text,
    pimg.license_id AS image_license_id,
    lic.short_name AS image_license_short_name,
    lic.name AS image_license_name,
    lic.url AS image_license_url,
    pimg.copyright AS image_copyright,
    pimg.created_by AS image_created_by,
    COALESCE(img_data.focus_x, pimg.focus_x) AS image_focus_x,
    COALESCE(img_data.focus_y, pimg.focus_y) AS image_focus_y,

    et_data.event_types,
    url_data.event_urls

FROM {{schema}}.event e
JOIN {{schema}}.organizer o ON o.id = e.organizer_id

-- Venue (fallback logic if event has venue_id)
LEFT JOIN {{schema}}.venue v ON v.id = e.venue_id

-- Space (fallback logic if event has space_id)
LEFT JOIN {{schema}}.space s ON s.id = e.space_id

-- Main image
LEFT JOIN LATERAL (
    SELECT TRUE AS has_main_image, eil.pluto_image_id AS id, 0 AS focus_x, 0 AS focus_y
    FROM {{schema}}.event_image_link eil
    WHERE eil.event_id = e.id AND eil.main_image = TRUE
    LIMIT 1
) img_data ON TRUE

-- Pluto image metadata
LEFT JOIN {{schema}}.pluto_image pimg ON pimg.id = img_data.id

-- License
LEFT JOIN {{schema}}.license_type lic
ON lic.license_id = pimg.license_id
AND lic.iso_639_1 = $2

-- Event types + genres
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
    JOIN {{schema}}.event_type et ON et.type_id = etl.type_id AND et.iso_639_1 = $2
    LEFT JOIN {{schema}}.genre_type gt ON gt.type_id = etl.genre_id AND gt.iso_639_1 = $2
    WHERE etl.event_id = e.id
) et_data ON TRUE


-- Event URLs
LEFT JOIN LATERAL (
SELECT jsonb_agg(
    jsonb_build_object(
        'id', eu.id,
        'url_type', eu.url_type,
        'url', eu.url,
        'title', eu.title
        )
    ) AS event_urls
    FROM {{schema}}.event_url eu
    WHERE eu.event_id = e.id
) url_data ON TRUE

WHERE e.id = $1