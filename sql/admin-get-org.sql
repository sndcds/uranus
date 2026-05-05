SELECT
    o.uuid::text AS uuid,
    o.name,
    o.description,
    o.legal_form,
    o.holding_org_uuid,
    o.nonprofit,
    o.contact_email,
    o.contact_phone,
    o.web_link,
    o.street,
    o.house_number,
    o.postal_code,
    o.city,
    o.state,
    o.country,
    o.address_addition,
    ST_X(o.point) AS lon,
    ST_Y(o.point) AS lat,
    img.images
FROM {{schema}}.organization o
JOIN {{schema}}.user_organization_link uol
ON uol.org_uuid = o.uuid AND uol.user_uuid = $2
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
    WHERE pil.context = 'organization'
    AND pil.context_uuid = o.uuid
) img ON true
WHERE o.uuid = $1