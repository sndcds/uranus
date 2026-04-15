WITH event_data AS (
    SELECT
        ed.uuid AS event_date_uuid,
        ed.event_uuid,
        ed.space_uuid,
        ed.start_date,
        ed.end
    FROM {{schema}}.event_date ed
    WHERE ed.event_uuid = $1
)
SELECT
    e.uuid AS event_uuid,
    e.org_uuid AS org_uuid,
    COALESCE(s.uuid, e.space_uuid) AS space_uuid,
    e.venue_uuid
FROM event_data ed
LEFT JOIN {{schema}}.event e ON ed.event_uuid = e.uuid
LEFT JOIN {{schema}}.space s ON ed.space_uuid = s.uuid