SELECT
    v.name AS name,
    v.city AS city,
    ST_X(v.point) AS lon,
    ST_Y(v.point) AS lat,
    v.web_link AS link,
    v.type AS type,
    vti.name AS type_name,
    COALESCE(e.event_count, 0) AS event_count

FROM {{schema}}.venue v
LEFT JOIN {{schema}}.venue_type_i18n vti ON vti.key = v.type AND vti.iso_639_1 = $1

LEFT JOIN (
    SELECT
        COALESCE(edp.venue_uuid, ep.venue_uuid) AS venue_uuid,
        COUNT(*) AS event_count
    FROM {{schema}}.event_date_projection edp
    JOIN {{schema}}.event_projection ep ON ep.event_uuid = edp.event_uuid
    GROUP BY COALESCE(edp.venue_uuid, ep.venue_uuid)
) e ON e.venue_uuid = v.uuid

GROUP BY v.uuid, vti.name, e.event_count