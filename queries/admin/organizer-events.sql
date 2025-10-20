WITH event_data AS (
    SELECT
        ed.id AS event_date_id,
        ed.event_id,
        ed.space_id,
        ed.start,
        ed.end,
        ed.entry_time,
        ed.duration,
        ed.accessibility_flags,
        ed.visitor_info_flags
    FROM {{schema}}.event_date ed
)
SELECT
    e.id AS event_id,
    e.title AS event_title,
    e.subtitle AS event_subtitle,
    e.organizer_id AS event_organizer_id,
    TO_CHAR(ed.start, 'YYYY-MM-DD') AS start_date,
    TO_CHAR(ed.start, 'HH24:MI') AS start_time,
    TO_CHAR(ed.end, 'YYYY-MM-DD') AS end_date,
    TO_CHAR(ed.end, 'HH24:MI') AS end_time,
    v.id AS venue_id,
    v.name AS venue_name,
    COALESCE(s.id, es.id) AS space_id,
    COALESCE(s.name, es.name) AS space_name,
    ST_X(v.wkb_geometry) AS venue_lon,
    ST_Y(v.wkb_geometry) AS venue_lat

FROM event_data ed
         JOIN {{schema}}.event e ON ed.event_id = e.id
         LEFT JOIN {{schema}}.space s ON ed.space_id = s.id
         LEFT JOIN {{schema}}.space es ON e.space_id = es.id
         JOIN {{schema}}.venue v ON COALESCE(s.venue_id, es.venue_id) = v.id
         JOIN {{schema}}.organizer o ON v.organizer_id = o.id

WHERE o.id = $1
AND ed.start::date >= $2
