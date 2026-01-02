SELECT
    oml.id AS member_id,
    u.id AS user_id,
    u.email_address AS email,
    u.user_name,
    COALESCE(u.display_name, u.first_name || ' ' || u.last_name, u.email_address) AS display_name,
    u.modified_at AS last_active_at,
    oml.created_at AS joined_at
FROM {{schema}}.organization_member_link oml
JOIN {{schema}}.user u ON u.id = oml.user_id
WHERE oml.organization_id = $1 AND oml.has_joined = TRUE
ORDER BY display_name