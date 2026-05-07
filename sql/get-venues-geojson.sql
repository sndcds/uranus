SELECT
    v.uuid::text,
    v.name,
    v.city,
    v.country,
    ST_X(v.point) AS lon,
    ST_Y(v.point) AS lat
FROM {{schema}}.venue v
WHERE v.point IS NOT NULL
AND ST_Within(v.point, ST_MakeEnvelope($1, $2, $3, $4, 4326))