SELECT
    v.id,
    v.name,
    v.description,
    v.type,
    v.organization_id,
    TO_CHAR(v.opened_at, 'YYYY-MM-DD') AS opened_at,
    TO_CHAR(v.closed_at, 'YYYY-MM-DD') AS closed_at,
    v.contact_email,
    v.contact_phone,
    v.website_link,
    v.street,
    v.house_number,
    v.postal_code,
    v.city,
    v.state,
    v.country,
    ST_X(v.geo_pos) AS lon,
    ST_Y(v.geo_pos) AS lat,
    img.images
FROM {{schema}}.venue v
JOIN {{schema}}.organization o ON o.id = v.organization_id
JOIN {{schema}}.user_organization_link uol ON uol.organization_id = o.id AND uol.user_id = $2
LEFT JOIN LATERAL (
    SELECT COALESCE(
        jsonb_object_agg(
            pil.identifier,
            jsonb_build_object(
                'id', pi.id,
                'focus_x', pi.focus_x,
                'focus_y', pi.focus_y,
                'alt', pi.alt_text,
                'copyright', pi.copyright,
                'license', pi.license_type,
                'creator', pi.creator_name
            )
        ),
        '{}'::jsonb
    ) AS images
    FROM {{schema}}.pluto_image_link pil
    JOIN {{schema}}.pluto_image pi
    ON pi.id = pil.pluto_image_id
    WHERE pil.context = 'venue'
    AND pil.context_id = v.id
) img ON true
WHERE v.id = $1
