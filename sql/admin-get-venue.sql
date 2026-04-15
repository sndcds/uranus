SELECT
    v.uuid,
    v.name,
    v.description,
    v.type,
    v.org_uuid,
    TO_CHAR(v.opened_at, 'YYYY-MM-DD') AS opened_at,
    TO_CHAR(v.closed_at, 'YYYY-MM-DD') AS closed_at,
    v.contact_email,
    v.contact_phone,
    v.web_link,
    v.street,
    v.house_number,
    v.postal_code,
    v.city,
    v.state,
    v.country,
    ST_X(v.point) AS lon,
    ST_Y(v.point) AS lat,
    img.images
FROM {{schema}}.venue v
JOIN {{schema}}.organization o ON o.uuid = v.org_uuid
JOIN {{schema}}.user_organization_link uol ON uol.org_uuid = o.uuid AND uol.user_uuid = $2::uuid
LEFT JOIN LATERAL (
    SELECT COALESCE(
        jsonb_object_agg(
            pil.identifier,
            jsonb_build_object(
                'uuid', pi.uuid,
                'focus_x', pi.focus_x,
                'focus_y', pi.focus_y,
                'alt', pi.alt_text,
                'copyright', pi.copyright,
                'license', pi.license,
                'creator', pi.creator_name
            )
        ),
        '{}'::jsonb
    ) AS images
    FROM {{schema}}.pluto_image_link pil
    JOIN {{schema}}.pluto_image pi
    ON pi.uuid = pil.pluto_image_uuid
    WHERE pil.context = 'venue'
    AND pil.context_uuid = v.uuid
) img ON true
WHERE v.uuid = $1::uuid
