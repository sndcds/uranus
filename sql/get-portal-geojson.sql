SELECT jsonb_build_object(
    'type', 'Feature',
    'geometry', ST_AsGeoJSON(geometry)::jsonb,
    'properties', jsonb_build_object(
        'uuid', uuid,
        'name', name,
        'description', description,
        'slug', slug
     )
) AS geojson
FROM {{schema}}.portal2
WHERE uuid = $1::uuid
    AND geometry IS NOT NULL