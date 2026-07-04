SELECT
    uuid
FROM {{schema}}.event_date
WHERE event_uuid = $1
    AND start_date = $2
    AND start_time = $3
LIMIT 1