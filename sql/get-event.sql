SELECT
    e.id AS event_id,
    e.release_status_id,
    e.title,
    e.subtitle,
    e.summary,
    e.description,
    e.summary,
    e.participation_info,
    e.meeting_point,
    e.languages,
    e.tags,

    o.id AS organization_id,
    o.name AS organization_name,
    o.website_url AS organization_url,

    l.id AS location_id,
    l.name AS location_name,
    l.street AS location_street,
    l.house_number AS location_house_number,
    l.city AS location_city,
    l.country_code AS location_country_code,
    l.state_code AS location_state_code,
    ST_X(l.wkb_pos) AS location_lon,
    ST_Y(l.wkb_pos) AS location_lat,

    -- Main image
    (image_data.id IS NOT NULL) AS has_main_image,
    image_data.id AS image_id,
    COALESCE(pimg.width, 0) AS image_width,
    COALESCE(pimg.height, 0) AS image_height,
    pimg.mime_type AS image_mime_type,
    pimg.alt_text AS image_alt_text,
    pimg.license_id AS image_license_id,
    lic.short_name AS image_license_short_name,
    lic.name AS image_license_name,
    lic.url AS image_license_url,
    pimg.copyright AS image_copyright,
    pimg.creator_name AS image_creator_name,
    COALESCE(pimg.focus_x, 0) AS image_focus_x,
    COALESCE(pimg.focus_y, 0) AS image_focus_y,

    -- Event types
    et_data.event_types,

    -- Event URLs
    url_data.event_urls

FROM {{schema}}.event e
JOIN {{schema}}.organization o ON o.id = e.organization_id

-- Venue (fallback logic if event has venue_id)
LEFT JOIN {{schema}}.venue v ON v.id = e.venue_id

-- Space (fallback logic if event has space_id)
LEFT JOIN {{schema}}.space s ON s.id = e.space_id

-- Location
LEFT JOIN {{schema}}.event_location l ON l.id = e.location_id


-- Main image (first non-null of the four direct columns)
LEFT JOIN LATERAL (
    SELECT COALESCE(e.image1_id, e.image2_id, e.image3_id, e.image4_id) AS id
) image_data ON TRUE

-- Pluto image metadata
LEFT JOIN {{schema}}.pluto_image pimg
ON pimg.id = image_data.id

-- License
LEFT JOIN {{schema}}.license_type lic
ON lic.license_id = pimg.license_id
AND lic.iso_639_1 = $2

-- Event types
LEFT JOIN LATERAL (
    SELECT COALESCE(
        jsonb_agg(DISTINCT jsonb_build_object(
            'type_id', etl.type_id,
            'type_name', et.name,
            'genre_id', COALESCE(gt.type_id, 0),
            'genre_name', gt.name
        )), '[]'::jsonb
    ) AS event_types
    FROM {{schema}}.event_type_link etl
    JOIN {{schema}}.event_type et
    ON et.type_id = etl.type_id
    AND et.iso_639_1 = $2
    LEFT JOIN {{schema}}.genre_type gt
    ON gt.type_id = etl.genre_id
    AND gt.iso_639_1 = $2
    WHERE etl.event_id = $1
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