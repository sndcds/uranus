SELECT
    e.id AS id,
    e.title,
    e.subtitle,
    e.description,
    e.teaser_text,
    e.participation_info,
    e.languages,
    e.tags,
    e.meeting_point,
    e.release_status_id,
    e.release_date,

    o.id AS organizer_id,
    o.name AS organizer_name,

    v.id AS venue_id,
    v.name AS venue_name,
    v.street AS venue_street,
    v.house_number AS venue_house_number,
    v.postal_code AS venue_postal_code,
    v.city AS venue_city,
    v.state_code AS venue_state_code,
    v.country_code AS venue_country_code,
    ST_X(v.wkb_geometry) AS venue_lon,
    ST_Y(v.wkb_geometry) AS venue_lat,

    space_data.id AS space_id,
    space_data.name AS space_name,
    space_data.total_capacity AS space_total_capacity,
    space_data.seating_capacity AS space_seating_capacity,
    space_data.building_level AS space_building_level,
    space_data.website_url AS space_url,

    -- Main image fields
    img_data.image_id,
    img_data.image_focus_x,
    img_data.image_focus_y,
    img_data.image_alt_text,
    img_data.image_copyright,
    img_data.image_created_by,
    img_data.image_license_id,

    -- Event types
    (
        SELECT jsonb_agg(jsonb_build_object(
                'type_id', etl.type_id,
                'type_name', et.name,
                'genre_id', COALESCE(gt.type_id, 0),
                'genre_name', gt.name
                         ))
        FROM {{schema}}.event_type_link etl
        JOIN {{schema}}.event_type et
ON et.type_id = etl.type_id AND et.iso_639_1 = $2
    LEFT JOIN {{schema}}.genre_type gt
    ON gt.type_id = etl.genre_id AND gt.iso_639_1 = $2
WHERE etl.event_id = e.id
    ) AS event_types,

-- Event URLs
    (
SELECT jsonb_agg(jsonb_build_object(
    'id', eu.id,
    'url_type', eu.url_type,
    'url', eu.url,
    'title', eu.title
    ))
FROM {{schema}}.event_url eu
WHERE eu.event_id = e.id
    ) AS event_urls

FROM {{schema}}.event e
    LEFT JOIN {{schema}}.organizer o ON e.organizer_id = o.id
    LEFT JOIN {{schema}}.venue v ON v.id = e.venue_id

    -- LATERAL join for space
    LEFT JOIN LATERAL (
    SELECT *
    FROM {{schema}}.space s2
    WHERE s2.id = e.space_id
    LIMIT 1
    ) space_data ON TRUE

    -- LATERAL join for main image
    LEFT JOIN LATERAL (
    SELECT
    pi.id AS image_id,
    pi.focus_x AS image_focus_x,
    pi.focus_y AS image_focus_y,
    pi.alt_text AS image_alt_text,
    pi.copyright AS image_copyright,
    pi.created_by AS image_created_by,
    pi.license_id AS image_license_id
    FROM {{schema}}.event_image_link eil
    JOIN {{schema}}.pluto_image pi ON pi.id = eil.pluto_image_id
    WHERE eil.event_id = e.id AND eil.main_image = TRUE
    LIMIT 1
    ) img_data ON TRUE

WHERE e.id = $1
    LIMIT 1;