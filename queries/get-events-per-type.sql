WITH event_data AS (
    SELECT
        ed.id AS event_date_id,
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
    COUNT(DISTINCT ed.event_date_id) AS count
FROM {{schema}}.event_type et
    LEFT JOIN {{schema}}.event_type_links etl
ON et.type_id = etl.type_id
    LEFT JOIN event_data ed
    ON ed.event_id = etl.event_id
WHERE et.iso_639_1 = $1
    {{conditions}}
GROUP BY et.type_id, et.name
HAVING COUNT(DISTINCT ed.event_date_id) > 0
    ),

    total_count AS (
SELECT COUNT(DISTINCT event_date_id) AS total
FROM event_data
    )

SELECT json_build_object(
               'total', (SELECT total FROM total_count),
               'types', (SELECT json_agg(ec ORDER BY ec.name) FROM event_counts ec)
       ) AS result;