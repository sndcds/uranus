SELECT COALESCE(
    jsonb_agg(
        DISTINCT jsonb_build_object(
            'type_id', etl.type_id,
            'type_name', et.name,
            'genre_id', COALESCE(gt.type_id, 0),
            'genre_name', gt.name
        )
    ),'[]'::jsonb
) AS event_types
FROM {{schema}}.event_type_link etl
JOIN {{schema}}.event_type et
ON et.type_id = etl.type_id
AND et.iso_639_1 = $2
LEFT JOIN {{schema}}.genre_type gt
ON gt.type_id = etl.genre_id
AND gt.iso_639_1 = $2
WHERE etl.event_id = $1