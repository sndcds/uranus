SELECT
    v.venue_uuid,
    v.venue_name,
    v.venue_city,
    v.venue_country,
    ST_X(v.venue_point) AS venue_lon,
    ST_Y(v.venue_point) AS venue_lat,
    COUNT(DISTINCT edp.event_uuid) AS event_count
FROM {{schema}}.event_date_projection edp

JOIN {{schema}}.event_projection ep
ON ep.event_uuid = edp.event_uuid

    LEFT JOIN LATERAL (
    SELECT
    COALESCE(edp.venue_uuid, ep.venue_uuid) AS venue_uuid,
    COALESCE(edp.venue_name, ep.venue_name) AS venue_name,
    COALESCE(edp.venue_city, ep.venue_city) AS venue_city,
    COALESCE(edp.venue_country, ep.venue_country) AS venue_country,
    COALESCE(edp.venue_point, ep.venue_point) AS venue_point
    ) v ON true

WHERE {{date_conditions}}
{{conditions}}

AND v.venue_uuid IS NOT NULL

GROUP BY
    v.venue_uuid,
    v.venue_name,
    v.venue_city,
    v.venue_country,
    v.venue_point

ORDER BY event_count DESC
{{limit}}