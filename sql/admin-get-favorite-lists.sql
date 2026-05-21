SELECT
    fl.uuid,
    fl.name,
    fl.description
FROM {{schema}}.favorite_list fl
JOIN {{schema}}.user_organization_link uol
    ON uol.org_uuid = fl.org_uuid
    AND uol.user_uuid = $2::uuid
WHERE fl.org_uuid = $1::uuid