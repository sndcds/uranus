SELECT
    p.uuid,
    p.name,
    p.description,
    uol.permissions
FROM {{schema}}.portal p

LEFT JOIN {{schema}}.user_organization_link uol
ON uol.org_uuid = p.org_uuid
AND uol.user_uuid = $2::uuid

WHERE p.org_uuid = $1::uuid
