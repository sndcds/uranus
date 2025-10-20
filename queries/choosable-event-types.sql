SELECT
    et.type_id AS type_id,
    et.name AS type_name
FROM {{schema}}.event_type et
WHERE et.iso_639_1 = $1
ORDER BY LOWER(et.name);