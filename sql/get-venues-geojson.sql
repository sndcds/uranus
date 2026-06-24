SELECT
    v.uuid::text,
    v.name,
    v.city,
    v.country,
    ST_X(v.point) AS lon,
    ST_Y(v.point) AS lat,
    pil.pluto_image_uuid::text AS logo_uuid
FROM {{schema}}.venue v
LEFT JOIN {{schema}}.pluto_image_link pil
    ON pil.context = 'venue'
        AND pil.context_uuid = v.uuid
        AND pil.identifier = 'main_logo'
WHERE v.point IS NOT NULL
    AND ST_Within(v.point, ST_MakeEnvelope($1, $2, $3, $4, 4326))