SELECT
    e.uuid AS event_uuid,
    e.release_status,
    e.content_iso_639_1,
    e.title,
    e.subtitle,
    e.description,
    e.summary,
    e.participation_info,
    e.online_link,
    e.registration_link,
    e.registration_email,
    e.registration_phone,
    TO_CHAR(e.registration_deadline, 'YYYY-MM-DD') AS registration_deadline,
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
    e.ticket_flags,
    e.ticket_link,
    e.visitor_info_flags,
    o.uuid AS org_uuid,
    o.name AS org_name,
    o.web_link AS org_link,
    org_logos,

    CASE
        WHEN main_image.uuid IS NULL THEN NULL
        ELSE jsonb_build_object(
            'uuid', main_image.uuid,
            'url', format('{{base_api_url}}/api/image/%s', main_image.uuid),
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
JOIN {{schema}}.organization o ON o.uuid = e.org_uuid

-- Venue (fallback logic if event has venue_uuid)
LEFT JOIN {{schema}}.venue v ON v.uuid = e.venue_uuid

-- Space (fallback logic if event has space_uuid)
LEFT JOIN {{schema}}.space s ON s.uuid = e.space_uuid

-- Main image: first pluto_image linked as 'main', include license info
LEFT JOIN LATERAL (
    SELECT
        pi.uuid,
        pi.alt_text,
        pi.creator_name,
        pi.copyright,
        COALESCE(lic.key, lic_fallback.key) AS license,
        COALESCE(lic.name, lic_fallback.name) AS license_name,
        COALESCE(lic.description, lic_fallback.description) AS license_description
    FROM {{schema}}.pluto_image_link pil
    JOIN {{schema}}.pluto_image pi ON pi.uuid = pil.pluto_image_uuid
    LEFT JOIN {{schema}}.license_i18n lic ON lic.key = pi.license AND lic.iso_639_1 = $2
    LEFT JOIN {{schema}}.license_i18n lic_fallback ON lic_fallback.key = 'all-rights-reserved' AND lic_fallback.iso_639_1 = $2
    WHERE pil.context = 'event' AND pil.context_uuid = e.uuid AND pil.identifier = 'main'
    LIMIT 1
) main_image ON TRUE

LEFT JOIN LATERAL (
    SELECT
        COALESCE (
            jsonb_object_agg(
                pil.identifier,
                jsonb_build_object(
                    'uuid', pi.uuid::text,
                    'url', CASE
                    WHEN pi.uuid IS NULL THEN NULL
                    ELSE format('{{base_api_url}}/api/image/%s', pi.uuid)
                    END
                )
            ),
        '{}'::jsonb
        ) AS org_logos
    FROM {{schema}}.pluto_image_link pil
    JOIN {{schema}}.pluto_image pi
    ON pi.uuid = pil.pluto_image_uuid
    WHERE pil.context = 'organization'
    AND pil.context_uuid = o.uuid
    AND pil.identifier = ANY(ARRAY['main_logo', 'dark_theme_logo', 'light_theme_logo'])
) org_logos ON TRUE

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
    WHERE etl.event_uuid = $1::uuid
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
    WHERE eu.event_uuid = e.uuid
) link_data ON TRUE

WHERE e.uuid = $1::uuid
AND e.release_status = ANY($3)