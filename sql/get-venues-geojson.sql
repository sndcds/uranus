SELECT
    v.uuid::text,
    v.type,
    v.name,
    v.street,
    v.house_number,
    v.city,
    v.country,
    vt.marker_style,
    ST_X(v.point) AS lon,
    ST_Y(v.point) AS lat,
    v.web_link,
    format('{{base_api_url}}/api/image/%s', pil.pluto_image_uuid::text) AS logo_url
FROM {{schema}}.venue v
LEFT JOIN {{schema}}.pluto_image_link pil
    ON pil.context = 'venue'
        AND pil.context_uuid = v.uuid
        AND pil.identifier = 'main_logo'
LEFT JOIN {{schema}}.venue_type vt ON vt.key = v.type
WHERE v.point IS NOT NULL
    AND ST_Within(v.point, ST_MakeEnvelope($1, $2, $3, $4, 4326))