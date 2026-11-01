WITH child_type_counts AS (
    SELECT parent_uuid, type, COUNT(*) AS count
    FROM {{schema}}.portal_geometry
    WHERE portal_uuid = $1::uuid
        AND parent_uuid IS NOT NULL
        AND type IS NOT NULL
    GROUP BY parent_uuid, type
), child_types AS (
    SELECT
        parent_uuid,
        jsonb_agg(
            jsonb_build_object('type', type, 'count', count)
            ORDER BY type
        ) AS child_types
    FROM child_type_counts
    GROUP BY parent_uuid
)
SELECT
    g.uuid::text,
    g.parent_uuid::text,
    COALESCE(g.name, ''),
    COALESCE(g.description, ''),
    COALESCE(g."order", 0),
    g.admin_level,
    g.type,
    ST_AsGeoJSON(g.geometry)::jsonb AS geometry,
    COALESCE(ct.child_types, '[]'::jsonb) AS child_types
FROM {{schema}}.portal2 p
LEFT JOIN {{schema}}.portal_geometry g
    ON g.portal_uuid = p.uuid
LEFT JOIN child_types ct
    ON ct.parent_uuid = g.uuid
WHERE p.uuid = $1::uuid
ORDER BY COALESCE(g."order", 0), g.name, g.uuid;
