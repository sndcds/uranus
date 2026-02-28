SELECT
    edp.event_date_id,
    edp.event_id,
    v.venue_id,
    v.venue_name,
    v.venue_city,
    v.venue_country,
    ST_X(v.venue_geo_pos) AS venue_lon,
    ST_Y(v.venue_geo_pos) AS venue_lat,
    ep.title,
    TO_CHAR(edp.start_date, 'YYYY-MM-DD') AS start_date,
    TO_CHAR(edp.start_time, 'HH24:MI') AS start_time
FROM {{schema}}.event_date_projection edp
JOIN {{schema}}.event_projection ep
ON ep.event_id = edp.event_id
LEFT JOIN LATERAL (
    SELECT
        COALESCE(edp.venue_id, ep.venue_id) AS venue_id,
        COALESCE(edp.venue_name, ep.venue_name) AS venue_name,
        COALESCE(edp.venue_city, ep.venue_city) AS venue_city,
        COALESCE(edp.venue_country, ep.venue_country) AS venue_country,
        COALESCE(edp.venue_geo_pos, ep.venue_geo_pos) AS venue_geo_pos
) v ON true
WHERE {{date_conditions}}
{{conditions}}
{{limit}}