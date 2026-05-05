INSERT INTO {{schema}}.organization_partner_request (
    from_user_uuid,
    from_org_uuid,
    to_org_uuid,
    message,
    status
)
SELECT
    $1::uuid,
    $2::uuid,
    $3::uuid,
    $4,
    'pending'
WHERE EXISTS (
    SELECT 1
    FROM {{schema}}.organization o
    WHERE o.uuid = $3::uuid
)