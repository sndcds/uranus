SELECT
    uol.permissions
FROM {{schema}}.event e
JOIN {{schema}}.user u ON u.uuid = $1::uuid
JOIN {{schema}}.organization o ON o.uuid = e.org_uuid
JOIN {{schema}}.user_organization_link uol ON uol.org_uuid = o.uuid AND uol.user_uuid =  u.uuid
WHERE e.uuid = $2::uuid