SELECT TRUE AS is_member, oml.user_id
FROM uranus.organization_member_link oml
WHERE id = $1 AND organization_id = $2 AND user_id = $3