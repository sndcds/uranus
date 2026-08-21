SELECT
    v.uuid::text,
    v.type,
    v.name,
    v.street,
    v.house_number,
    v.city,
    v.country,
    v.scope,
    vt.marker_style,
    ST_AsGeoJSON(v.point) AS point,
    ST_AsGeoJSON(v.building) AS building,
    v.web_link,
    CASE
        WHEN pil.pluto_image_uuid IS NOT NULL
            THEN format(
                '{{base_api_url}}/api/image/%s',
                pil.pluto_image_uuid::text
             )
        ELSE NULL
    END AS logo_url

FROM {{schema}}.venue v

LEFT JOIN {{schema}}.pluto_image_link pil
    ON pil.context = 'venue'
        AND pil.context_uuid = v.uuid
        AND pil.identifier = 'main_logo'

LEFT JOIN {{schema}}.venue_type vt ON vt.key = v.type

WHERE v.point IS NOT NULL
    AND ST_Within(v.point, ST_MakeEnvelope($1, $2, $3, $4, 4326))
    AND (
        cardinality($5::text[]) = 0
        OR v.scope = ANY($5::text[])
    )