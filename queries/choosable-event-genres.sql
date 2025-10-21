SELECT
    gt.type_id AS genre_id,
    gt.name AS genre_name
FROM {{schema}}.genre_type gt
WHERE gt.event_type_id = $1
AND gt.iso_639_1 = $2
ORDER BY LOWER(gt.name)