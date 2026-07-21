SELECT
    v.uuid,
    v.org_uuid,

    v.type,
    vt.marker_style,
    vti.name AS type_name,
    vti.description AS type_description,

    v.name,
    v.description,
    v.summary,

    v.contact_email,
    v.contact_phone,
    v.web_link,

    v.street,
    v.house_number,
    v.postal_code,
    v.city,
    v.state,
    v.country,

    ST_Y(v.point)::text AS lat,
    ST_X(v.point)::text AS lon,

    COALESCE(to_char(v.opened_at, 'YYYY-MM-DD'), '') AS opened_at,
    COALESCE(to_char(v.closed_at, 'YYYY-MM-DD'), '') AS closed_at,

    v.ticket_info,
    v.ticket_link,
    v.opening_hours,

    v.accessibility_flags,
    v.accessibility_summary,

    v.content_iso_639_1,

    v.slug,

    images.logos,
    images.photos

FROM {{schema}}.venue v

LEFT JOIN {{schema}}.venue_type vt
    ON vt.key = v.type

LEFT JOIN {{schema}}.venue_type_i18n vti
    ON vti.key = v.type
    AND vti.iso_639_1 = $1

LEFT JOIN (
    SELECT
    context_uuid,

    jsonb_strip_nulls(
        jsonb_build_object(
            'main',  (array_agg(pluto_image_uuid) FILTER (WHERE identifier = 'main_logo'))[1],
            'dark',  (array_agg(pluto_image_uuid) FILTER (WHERE identifier = 'dark_theme_logo'))[1],
            'light', (array_agg(pluto_image_uuid) FILTER (WHERE identifier = 'light_theme_logo'))[1]
        )
    ) AS logos,

    jsonb_strip_nulls(
        jsonb_build_object(
            'main',      (array_agg(pluto_image_uuid) FILTER (WHERE identifier = 'main_photo'))[1],
            'gallery_1', (array_agg(pluto_image_uuid) FILTER (WHERE identifier = 'gallery_photo_1'))[1],
            'gallery_2', (array_agg(pluto_image_uuid) FILTER (WHERE identifier = 'gallery_photo_2'))[1],
            'gallery_3', (array_agg(pluto_image_uuid) FILTER (WHERE identifier = 'gallery_photo_3'))[1]
        )
    ) AS photos

    FROM {{schema}}.pluto_image_link
    WHERE context = 'venue'
    GROUP BY context_uuid
) images
    ON images.context_uuid = v.uuid

WHERE TRUE
    {{conditions}}

ORDER BY
    v.name;

{{limit}}