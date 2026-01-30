SELECT
    pi.id,
    pil.identifier,
    pi.focus_x,
    pi.focus_y,
    pi.alt_text AS alt,
    pi.copyright,
    pi.creator_name AS creator,
    pi.license_id AS license
FROM {{schema}}.pluto_image_link pil
JOIN {{schema}}.pluto_image pi ON pi.id = pil.pluto_image_id
WHERE pil.context = 'event' AND pil.context_id = $1