SELECT
    e.id AS event_id,
    e.release_status,
    e.title,
    e.subtitle,
    e.description,
    e.summary,
    e.participation_info,
    e.meeting_point,
    e.languages,
    e.tags,
    e.max_attendees,
    e.min_age,
    e.max_age,
    e.currency,
    e.price_type,
    e.min_price,
    e.max_price,
    e.visitor_info_flags,
    o.id AS organization_id,
    o.name AS organization_name,
    o.website_link AS organization_link,

    CASE
        WHEN main_image.id IS NULL THEN NULL
        ELSE jsonb_build_object(
            'id', main_image.id,
            'url', format('{{base_api_url}}/api/image/%s', main_image.id),
            'alt', main_image.alt_text,
            'creator', main_image.creator_name,
            'copyright', main_image.copyright,
            'license', main_image.license,
            'license_name', main_image.license_name,
            'license_description', main_image.license_description
        )
        END AS image,

    et_data.event_types,
    link_data.event_links

FROM {{schema}}.event e
JOIN {{schema}}.organization o ON o.id = e.organization_id

-- Venue (fallback logic if event has venue_id)
LEFT JOIN {{schema}}.venue v ON v.id = e.venue_id

-- Space (fallback logic if event has space_id)
LEFT JOIN {{schema}}.space s ON s.id = e.space_id

-- Main image: first pluto_image linked as 'main', include license info
LEFT JOIN LATERAL (
    SELECT
        pi.id,
        pi.alt_text,
        pi.creator_name,
        pi.copyright,
        COALESCE(lic.key, lic_fallback.key) AS license,
        COALESCE(lic.name, lic_fallback.name) AS license_name,
        COALESCE(lic.description, lic_fallback.description) AS license_description
    FROM {{schema}}.pluto_image_link pil
    JOIN {{schema}}.pluto_image pi ON pi.id = pil.pluto_image_id
    LEFT JOIN {{schema}}.license_i18n lic ON lic.key = pi.license AND lic.iso_639_1 = $2
    LEFT JOIN {{schema}}.license_i18n lic_fallback ON lic_fallback.key = 'all-rights-reserved' AND lic_fallback.iso_639_1 = $2
    WHERE pil.context = 'event' AND pil.context_id = e.id AND pil.identifier = 'main'
    LIMIT 1
) main_image ON TRUE

-- Event types
LEFT JOIN LATERAL (
    SELECT COALESCE(
        jsonb_agg(DISTINCT jsonb_build_object(
            'type_id', etl.type_id,
            'type_name', et.name,
            'genre_id', COALESCE(gt.genre_id, 0),
            'genre_name', gt.name
        )), '[]'::jsonb
    ) AS event_types
    FROM {{schema}}.event_type_link etl
    JOIN {{schema}}.event_type et
    ON et.type_id = etl.type_id
    AND et.iso_639_1 = $2
    LEFT JOIN {{schema}}.genre_type gt
    ON gt.genre_id = etl.genre_id
    AND gt.iso_639_1 = $2
    WHERE etl.event_id = $1
) et_data ON TRUE

-- Event URLs
LEFT JOIN LATERAL (
    SELECT jsonb_agg(
        jsonb_build_object(
            'label', eu.label,
            'type', eu.type,
            'url', eu.url
        )
    ) AS event_links
    FROM {{schema}}.event_link eu
    WHERE eu.event_id = e.id
) link_data ON TRUE

WHERE e.id = $1