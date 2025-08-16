SELECT
    i.id,
    i.pluto_image_id,
    pi.file_name,
    pi.width,
    pi.height,
    pi.mime_type,
    pi.alt_text,
    i.caption,
    pi.copyright,
    pi.license,
    i.focus_x,
    i.focus_y
FROM {{schema}}.event_image_links eil
JOIN {{schema}}.image i ON i.id = eil.image_id
JOIN {{schema}}.pluto_image pi ON pi.id = i.pluto_image_id
WHERE eil.event_id = $1;