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
    et.type_id,
    et.name AS type_name,
    COUNT(DISTINCT ed.event_id) AS event_count
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
SELECT jsonb_agg(
               jsonb_build_object(
                       'type_id', type_id,
                       'type_name', type_name,
                       'count', event_count
               ) ORDER BY type_name
       ) AS event_type_counts
FROM event_counts