SELECT
    a.src_org_uuid,
    a.dst_org_uuid,
    srco.name AS src_org_name,
    dsto.name AS dst_org_name,
    a.permissions,

    EXISTS (
        SELECT 1
        FROM {{schema}}.user_organization_link uo
        WHERE uo.user_uuid = $1::uuid
        AND uo.org_uuid = a.src_org_uuid
    ) AS src_access,

    EXISTS (
        SELECT 1
        FROM {{schema}}.user_organization_link uo
        WHERE uo.user_uuid = $1::uuid
        AND uo.org_uuid = a.dst_org_uuid
    ) AS dst_access

FROM {{schema}}.organization_access_grants a
JOIN {{schema}}.organization srco ON srco.uuid = a.src_org_uuid
JOIN {{schema}}.organization dsto ON dsto.uuid = a.dst_org_uuid

WHERE a.src_org_uuid IN (
    SELECT org_uuid FROM {{schema}}.user_organization_link WHERE user_uuid = $1::uuid
)
OR a.dst_org_uuid IN (
    SELECT org_uuid FROM {{schema}}.user_organization_link WHERE user_uuid = $1::uuid
)