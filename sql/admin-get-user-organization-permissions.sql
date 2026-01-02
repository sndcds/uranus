SELECT uol.permissions::bigint AS perm
FROM {{schema}}.user_organization_link uol
JOIN {{schema}}.organization_member_link oml
ON uol.user_id = oml.user_id
AND uol.organization_id = oml.organization_id
WHERE uol.user_id = $1
AND uol.organization_id = $2
AND oml.has_joined = true
LIMIT 1