SELECT jsonb_build_object(
    'type', 'Feature',
    'geometry', ST_AsGeoJSON(pg.geometry)::jsonb,
    'properties', jsonb_build_object(
        'uuid', p.uuid,
        'name', p.name,
        'description', p.description,
        'slug', p.slug
    )
) AS geojson

FROM {{schema}}.portal2 p

LEFT JOIN {{schema}}.portal_geometry pg
    ON (
        (
            $2::uuid IS NULL
                AND pg.portal_uuid = p.uuid
                AND pg.parent_uuid IS NULL
        )
        OR
        (
            $2::uuid IS NOT NULL
                AND pg.uuid = $2::uuid
        )
    )

WHERE p.uuid = $1::uuid
    AND pg.geometry IS NOT NULL;