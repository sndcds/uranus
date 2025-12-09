SELECT
COALESCE(uvl.permissions, 0) | COALESCE(uol.permissions, 0) AS combined_permissions
FROM {{schema}}.venue v
LEFT JOIN {{schema}}.user_venue_link uvl
ON uvl.venue_id = v.id
AND uvl.user_id = $1
LEFT JOIN {{schema}}.user_organizer_link uol
ON uol.organizer_id = v.organizer_id
AND uol.user_id = $1
WHERE v.organizer_id = $2
AND v.id = $3
LIMIT 1