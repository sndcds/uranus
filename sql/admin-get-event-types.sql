SELECT
    et.type_id,
    et.name AS type_name,
    COALESCE(gt.genre_id, 0) AS genre_id,
    gt.name AS genre_name
FROM {{schema}}.event_type_link etl
JOIN {{schema}}.event_type et ON et.type_id = etl.type_id AND et.iso_639_1 = $2
LEFT JOIN {{schema}}.genre_type gt ON gt.genre_id = etl.genre_id AND gt.iso_639_1 = $2
WHERE etl.event_id = $1
ORDER BY et.name, gt.name