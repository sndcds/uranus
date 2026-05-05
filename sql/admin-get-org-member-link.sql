SELECT
    oml.org_uuid,
    oml.user_uuid,
    oml.has_joined,
    oml.invited_by_user_uuid,
    oml.invited_at,
    oml.created_at,
    oml.modified_at
FROM {{schema}}.organization_member_link oml
WHERE oml.user_uuid = $1::uuid