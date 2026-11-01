SELECT
    g.uuid::text,
    g.parent_uuid::text,
    COALESCE(g.name, ''),
    COALESCE(g.description, ''),
    COALESCE(g."order", 0),
    g.admin_level,
    g.type,
    ST_AsGeoJSON(g.geometry)::jsonb AS geometry
FROM {{schema}}.portal2 p
LEFT JOIN {{schema}}.portal_geometry g
    ON g.portal_uuid = p.uuid
WHERE p.uuid = $1::uuid
ORDER BY COALESCE(g."order", 0), g.name, g.uuid;