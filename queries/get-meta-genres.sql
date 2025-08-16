SELECT
    et.name AS type_name,
    et.type_id AS type_id,
    gt.name AS genre_name,
    gt.type_id AS genre_id
FROM {{schema}}.event_type et
    LEFT JOIN {{schema}}.genre_type gt
        ON gt.event_type_id = et.type_id
        AND gt.iso_639_1 = $1
WHERE et.iso_639_1 = $1
ORDER BY LOWER(et.name), LOWER(gt.name);