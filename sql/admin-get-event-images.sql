SELECT
    pi.uuid,
    pil.identifier,
    pi.focus_x,
    pi.focus_y,
    pi.alt_text AS alt,
    pi.copyright,
    pi.creator_name AS creator,
    pi.license AS license,
    pi.width,
    pi.height
FROM {{schema}}.pluto_image_link pil
JOIN {{schema}}.pluto_image pi ON pi.uuid = pil.pluto_image_uuid
WHERE pil.context = 'event' AND pil.context_uuid = $1
ORDER BY pil.identifier, pi.uuid
