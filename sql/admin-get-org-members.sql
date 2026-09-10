SELECT
    u.uuid AS user_uuid,
    u.email AS email,
    u.username,
    COALESCE(u.display_name, u.first_name || ' ' || u.last_name, u.email) AS display_name,
    u.modified_at AS last_active_at,
    oml.created_at AS joined_at,
    NOT EXISTS (SELECT 1 FROM {{schema}}.user_organization_link uol WHERE uol.org_uuid = oml.org_uuid AND uol.user_uuid = oml.user_uuid AND uol.permissions <> 0) AS permissions_missing
FROM {{schema}}.organization_member_link oml
JOIN {{schema}}.user u ON u.uuid = oml.user_uuid
WHERE oml.org_uuid = $1 AND oml.has_joined = TRUE
ORDER BY display_name