WITH event_data AS (
    SELECT
        ed.event_id
    FROM {{schema}}.event_date ed
    JOIN {{schema}}.event e
ON ed.event_id = e.id
    AND e.release_status_id >= 2
    {{event-date-conditions}}
    ),
    event_counts AS (
SELECT
    et.type_id AS id,
    et.name,
    COUNT(DISTINCT ed.event_id) AS count
FROM {{schema}}.event_type et
    LEFT JOIN {{schema}}.event_type_links etl
ON et.type_id = etl.type_id
    LEFT JOIN event_data ed
    ON ed.event_id = etl.event_id
WHERE et.iso_639_1 = $1
    {{conditions}}
    {{limit}}
GROUP BY et.type_id, et.name
HAVING COUNT(DISTINCT ed.event_id) > 0
    )
SELECT *
FROM event_counts
ORDER BY name