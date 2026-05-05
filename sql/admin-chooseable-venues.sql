SELECT
    src_o.uuid AS org_uuid,
    src_o.name AS org_name,
    v.uuid AS venue_uuid,
    v.name AS venue_name,
    s.uuid AS space_uuid,
    s.name AS space_name,
    v.city,
    v.country,
    oag.permissions AS permissions
FROM {{schema}}.organization o
JOIN {{schema}}.organization_access_grants oag ON oag.dst_org_uuid = o.uuid
JOIN {{schema}}.organization src_o ON src_o.uuid = oag.src_org_uuid
LEFT JOIN {{schema}}.venue v ON v.org_uuid = src_o.uuid
LEFT JOIN {{schema}}.space s ON s.venue_uuid = v.uuid
WHERE o.uuid = $1::uuid
AND (oag.permissions & $2::bigint) = $2::bigint

UNION ALL

SELECT
    o.uuid AS org_uuid,
    o.name AS org_name,
    v.uuid AS venue_uuid,
    v.name AS venue_name,
    s.uuid AS space_uuid,
    s.name AS space_name,
    v.city,
    v.country,
    (0xffffffff::bigint) AS permissions
FROM {{schema}}.organization o
LEFT JOIN {{schema}}.venue v ON v.org_uuid = o.uuid
LEFT JOIN {{schema}}.space s ON s.venue_uuid = v.uuid
WHERE o.uuid = $1::uuid

ORDER BY org_name, venue_name, space_name
