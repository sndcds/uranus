SELECT
    e.id AS event_id,
    e.title,
    e.subtitle,
    e.description,
    e.summary,
    e.participation_info,
    e.languages,
    e.tags,
    e.meeting_point,
    e.release_status_id,
    TO_CHAR(e.release_date, 'YYYY-MM-DD'),

    e.min_age,
    e.max_age,
    e.max_attendees,
    e.price_type_id,
    e.min_price,
    e.max_price,
    e.ticket_advance,
    e.ticket_required,
    e.registration_required,
    e.currency_code,
    COALESCE(cu.name, '-') AS currency_name,
    e.occasion_type_id,

    e.online_event_url,
    e.source_url,

    e.image1_id,
    e.image2_id,
    e.image3_id,
    e.image4_id,
    e.image_some_16_9_id,
    e.image_some_4_5_id,
    e.image_some_9_16_id,
    e.image_some_1_1_id,

    e.custom,
    e.style,

    o.id AS organization_id,
    o.name AS organization_name,

    v.id AS venue_id,
    v.name AS venue_name,
    v.street AS venue_street,
    v.house_number AS venue_house_number,
    v.postal_code AS venue_postal_code,
    v.city AS venue_city,
    v.country_code AS venue_country_code,
    v.state_code AS venue_state_code,
    ST_X(v.wkb_pos) AS venue_lon,
    ST_Y(v.wkb_pos) AS venue_lat,

    space_data.id AS space_id,
    space_data.name AS space_name,
    space_data.total_capacity AS space_total_capacity,
    space_data.seating_capacity AS space_seating_capacity,
    space_data.building_level AS space_building_level,
    space_data.website_url AS space_url,

    el.name AS location_name,
    el.street AS location_street,
    el.house_number AS location_house_number,
    el.postal_code AS location_postal_code,
    el.city AS location_city,
    el.country_code AS location_country_code,
    el.state_code AS location_state_code,

    COALESCE(
        (
            SELECT jsonb_agg(
                jsonb_build_object(
                    'type_id', etl.type_id,
                    'type_name', et.name,
                    'genre_id', COALESCE(gt.type_id, 0),
                    'genre_name', gt.name
                )
                ORDER BY et.name, gt.name
            )
            FROM {{schema}}.event_type_link etl
            JOIN {{schema}}.event_type et
            ON et.type_id = etl.type_id AND et.iso_639_1 = $2
            LEFT JOIN {{schema}}.genre_type gt
            ON gt.type_id = etl.genre_id AND gt.iso_639_1 = $2
            WHERE etl.event_id = e.id
        ), '[]'::jsonb
    ) AS event_types,

    -- Event URLs
    (
        SELECT jsonb_agg(
            jsonb_build_object(
               'id', eu.id,
               'url_type', eu.url_type,
               'url', eu.url,
               'title', eu.title
            )
        )
        FROM {{schema}}.event_url eu
        WHERE eu.event_id = e.id
    ) AS event_urls

FROM {{schema}}.event e
LEFT JOIN {{schema}}.organization o ON e.organization_id = o.id
LEFT JOIN {{schema}}.venue v ON v.id = e.venue_id
LEFT JOIN {{schema}}.event_location el ON e.location_id = el.id
LEFT JOIN {{schema}}.currency cu ON cu.code = e.currency_code AND cu.iso_639_1 = $2

LEFT JOIN {{schema}}.user_organization_link uol
ON uol.organization_id = o.id
AND uol.user_id = $3

LEFT JOIN {{schema}}.user_event_link uel
ON uel.event_id = e.id
AND uel.user_id = $3

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
    pi.creator_name AS image_creator_name,
    pi.license_id AS image_license_id
    FROM {{schema}}.pluto_image pi WHERE pi.id = e.image1_id
) image_data ON TRUE

WHERE ((uol.permissions & $4) <> 0 OR (uel.permissions & $4) <> 0)
AND e.id = $1
LIMIT 1