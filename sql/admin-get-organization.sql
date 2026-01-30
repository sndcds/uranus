SELECT
    o.name,
    o.description,
    o.legal_form_id,
    o.holding_organization_id,
    o.nonprofit,
    o.contact_email,
    o.contact_phone,
    o.website_url,
    o.street,
    o.house_number,
    o.postal_code,
    o.city,
    o.state,
    o.country,
    o.address_addition,
    ST_X(o.wkb_pos) AS lon,
    ST_Y(o.wkb_pos) AS lat,
    img.images
FROM {{schema}}.organization o
JOIN {{schema}}.user_organization_link uol
ON uol.organization_id = o.id AND uol.user_id = $2
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
                'license', pi.license_id,
                'creator', pi.creator_name
            )
        ),
        '{}'::jsonb
    ) AS images
    FROM {{schema}}.pluto_image_link pil
    JOIN {{schema}}.pluto_image pi
    ON pi.id = pil.pluto_image_id
    WHERE pil.context = 'organization'
    AND pil.context_id = o.id
) img ON true
WHERE o.id = $1