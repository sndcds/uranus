SELECT
    oml.organization_id,
    oml.user_id,
    oml.has_joined,
    oml.invited_by_user_id,
    oml.invited_at,
    oml.created_at,
    oml.modified_at
FROM {{schema}}.organization_member_link oml
WHERE oml.id = $1