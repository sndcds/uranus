SELECT (uol.permissions::bigint & $3::bigint) AS perm
FROM {{schema}}.user_organizer_link uol
JOIN {{schema}}.organizer_member_link oml
ON uol.user_id = oml.user_id
AND uol.organizer_id = oml.organizer_id
WHERE uol.user_id = $1
AND uol.organizer_id = $2
AND oml.has_joined = true
LIMIT 1