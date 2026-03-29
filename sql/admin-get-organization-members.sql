SELECT
    u.uuid AS user_uuid,
    u.email AS email,
    u.username,
    COALESCE(u.display_name, u.first_name || ' ' || u.last_name, u.email) AS display_name,
    u.modified_at AS last_active_at,
    oml.created_at AS joined_at
FROM {{schema}}.organization_member_link oml
JOIN {{schema}}.user u ON u.uuid = oml.user_uuid
WHERE oml.org_uuid = $1 AND oml.has_joined = TRUE
ORDER BY display_name