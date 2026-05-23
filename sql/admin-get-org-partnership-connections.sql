SELECT
    a.src_org_uuid,
    a.dst_org_uuid,
    srco.name AS src_org_name,
    dsto.name AS dst_org_name,
    a.permissions,

    EXISTS (
        SELECT 1
        FROM uranus.user_organization_link uo
        WHERE uo.user_uuid = $2::uuid
            AND uo.org_uuid = a.src_org_uuid
            AND (uo.permissions & $3::bigint) <> 0
    ) AS has_src_access,

    EXISTS (
        SELECT 1
        FROM uranus.user_organization_link uo
        WHERE uo.user_uuid = $2::uuid
            AND uo.org_uuid = a.dst_org_uuid
            AND (uo.permissions & $3::bigint) <> 0
    ) AS has_dst_access

FROM uranus.organization_access_grants a
    JOIN uranus.organization srco ON srco.uuid = a.src_org_uuid
    JOIN uranus.organization dsto ON dsto.uuid = a.dst_org_uuid

WHERE (a.src_org_uuid = $1::uuid OR a.dst_org_uuid = $1::uuid)