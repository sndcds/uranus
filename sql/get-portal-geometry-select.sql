SELECT
    g.name,
    COALESCE(g."order", 0),
    g.group_id,
    g.admin_level,
    g.uuid::text AS geometry_uuid
FROM {{schema}}.portal2 p
LEFT JOIN {{schema}}.portal_geometry g
    ON g.portal_uuid = p.uuid
    AND ($2::integer IS NULL OR g.group_id = $2::integer)
WHERE p.uuid = $1::uuid
ORDER BY COALESCE(g."order", 0), g.name, g.uuid;
