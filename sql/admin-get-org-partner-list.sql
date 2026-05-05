SELECT
    o.uuid,
    o.name,
    g.permissions,
    CASE
        WHEN g.dst_org_uuid = $1::uuid
            THEN 'incoming'
        ELSE 'outgoing'
        END AS direction
FROM {{schema}}.organization_access_grants g
JOIN {{schema}}.organization o
ON o.uuid = CASE
    WHEN g.src_org_uuid = $1::uuid
        THEN g.dst_org_uuid
    ELSE g.src_org_uuid
END
WHERE g.src_org_uuid = $1::uuid
OR g.dst_org_uuid = $1::uuid