INSERT INTO {{schema}}.organization_member_link
(org_uuid, user_uuid, accept_token, invited_at, invited_by_user_uuid)
VALUES ($1::uuid, $2::uuid, $3, CURRENT_TIMESTAMP, $4::uuid)
