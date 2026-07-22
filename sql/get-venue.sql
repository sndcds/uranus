SELECT
    v.uuid,
    v.name,
    v.type,
    vti.name AS type_name,
    vti.description AS type_description,
    TO_CHAR(v.opened_at, 'YYYY-MM-DD') AS opened_at,
    TO_CHAR(v.closed_at, 'YYYY-MM-DD') AS closed_at,
    v.summary,
    v.description,
    v.street,
    v.house_number,
    v.postal_code,
    v.city,
    v.country,
    v.state,
    v.contact_email,
    v.contact_phone,
    v.web_link,
    v.ticket_link,
    v.ticket_info,
    ST_X(v.point) AS lon,
    ST_Y(v.point) AS lat,
    v.accessibility_flags,
    v.accessibility_summary,
    v.org_uuid,
    o.name AS org_name,
    o.web_link AS org_web_link,
    o.city AS org_city,
    o.country AS org_country,

    COALESCE(
        json_agg(
            json_build_object(
                'uuid', s.uuid,
                'name', s.name,
                'total_capacity', s.total_capacity,
                'seating_capacity', s.seating_capacity,
                'building_level', s.building_level,
                'web_link', s.web_link,
                'description', s.description,
                'area_sqm', s.area_sqm,
                'space_type', s.space_type,
                'space_type_name', sti.name,
                'space_type_description', sti.description
            )
        ) FILTER (WHERE s.uuid IS NOT NULL),
        '[]'
    ) AS spaces,
    logos.logos,
    images.images

FROM {{schema}}.venue v

LEFT JOIN {{schema}}.organization o
    ON o.uuid = v.org_uuid

LEFT JOIN {{schema}}.space s
    ON s.venue_uuid = v.uuid

LEFT JOIN {{schema}}.venue_type vt
    ON vt.key = v.type

LEFT JOIN {{schema}}.venue_type_i18n vti
    ON vti.key = vt.key
        AND vti.iso_639_1 = $2

LEFT JOIN {{schema}}.space_type st
    ON st.key = s.space_type

LEFT JOIN {{schema}}.space_type_i18n sti
    ON sti.key = st.key
        AND sti.iso_639_1 = $2

-- Images
LEFT JOIN LATERAL (
    SELECT COALESCE(
        jsonb_object_agg(
            pil.identifier,
            jsonb_build_object(
                'uuid', pi.uuid,
                'identifier', pil.identifier,
                'url', format('{{base_api_url}}/api/image/%s', pi.uuid),
                'alt', pi.alt_text,
                'width', pi.width,
                'height', pi.height,
                'creator', pi.creator_name,
                'copyright', pi.copyright,
                'license', COALESCE(lic.key, lic_fallback.key),
                'license_name', COALESCE(lic.name, lic_fallback.name),
                'license_description', COALESCE(lic.description, lic_fallback.description)
            )
        ),
       '{}'::jsonb
    ) AS images

    FROM {{schema}}.pluto_image_link pil

    LEFT JOIN {{schema}}.pluto_image pi
        ON pi.uuid = pil.pluto_image_uuid

    LEFT JOIN {{schema}}.license_i18n lic
        ON lic.key = pi.license
            AND lic.iso_639_1 = $2

    LEFT JOIN {{schema}}.license_i18n lic_fallback
        ON lic_fallback.key = 'all-rights-reserved'
            AND lic_fallback.iso_639_1 = $2

    WHERE pil.context = 'venue'
        AND pil.context_uuid = v.uuid
        AND pil.identifier = ANY(
            ARRAY[
                'main_photo',
                'gallery1_photo',
                'gallery2_photo',
                'gallery3_photo'
            ]
        )
) images ON TRUE

-- Logos
LEFT JOIN LATERAL (
    SELECT COALESCE(
        jsonb_object_agg(
            pil.identifier,
            jsonb_build_object(
                'uuid', pi.uuid,
                'identifier', pil.identifier,
                'url', format('{{base_api_url}}/api/image/%s', pi.uuid)
            )
        ),
        '{}'::jsonb
    ) AS logos

    FROM {{schema}}.pluto_image_link pil

    JOIN {{schema}}.pluto_image pi
        ON pi.uuid = pil.pluto_image_uuid

    WHERE pil.context = 'venue'
        AND pil.context_uuid = v.uuid
        AND pil.identifier = ANY(
        ARRAY[
            'main_logo',
            'dark_theme_logo',
            'light_theme_logo'
        ]
    )

) logos ON TRUE

WHERE v.uuid = $1::uuid

GROUP BY
    v.uuid,
    o.uuid,
    vti.name,
    vti.description,
    logos.logos,
    images.images