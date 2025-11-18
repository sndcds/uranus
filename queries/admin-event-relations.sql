WITH event_data AS (
    SELECT
        ed.id AS event_date_id,
        ed.event_id,
        ed.space_id,
        ed.start_date,
        ed.end
    FROM {{schema}}.event_date ed
WHERE ed.event_id = $1
    )
SELECT
    e.id AS event_id,
    e.organizer_id AS event_organizer_id,
    COALESCE(s.id, e.space_id) AS space_id,
    e.venue_id
FROM event_data ed
         LEFT JOIN {{schema}}.event e ON ed.event_id = e.id
    LEFT JOIN {{schema}}.space s ON ed.space_id = s.id;