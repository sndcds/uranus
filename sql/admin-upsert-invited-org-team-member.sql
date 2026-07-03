INSERT INTO {{schema}}.organization_member_link AS oml (
    org_uuid,
    user_uuid,
    accept_token,
    invited_by_user_uuid
)
VALUES ($1::uuid, $2::uuid, $3, $4::uuid)
ON CONFLICT (org_uuid, user_uuid)
    DO UPDATE
    SET
        accept_token = EXCLUDED.accept_token,
        invited_by_user_uuid = EXCLUDED.invited_by_user_uuid,
        invited_at = NOW()
    WHERE NOT oml.has_joined