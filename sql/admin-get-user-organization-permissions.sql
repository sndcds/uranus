SELECT uol.permissions::bigint AS perm
FROM {{schema}}.user_organization_link uol
JOIN {{schema}}.organization_member_link oml
ON uol.user_uuid = oml.user_uuid
AND uol.org_uuid = oml.org_uuid
WHERE uol.user_uuid = $1
AND uol.org_uuid = $2
AND oml.has_joined = true
LIMIT 1