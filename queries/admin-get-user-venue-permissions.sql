WITH user_venue_perm AS (
    SELECT uv.venue_id, uv.permissions
    FROM {{schema}}.user_venue_link uv
    WHERE uv.user_id = $1
),
user_org_perm AS (
    SELECT v.id AS venue_id, uo.permissions
    FROM {{schema}}.venue v
    JOIN {{schema}}.user_organizer_link uo
    ON uo.organizer_id = v.organizer_id
    WHERE uo.user_id = $1
    )
SELECT
    COALESCE(uv.permissions, 0) | COALESCE(uo.permissions, 0) AS combined_permissions
FROM {{schema}}.venue v
    LEFT JOIN user_venue_perm uv ON uv.venue_id = v.id
    LEFT JOIN user_org_perm uo ON uo.venue_id = v.id
WHERE v.id = $2