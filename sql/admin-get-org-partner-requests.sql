SELECT
    to_o.uuid AS org_uuid,
    to_o.name AS org_name,
    opr.created_at,
    opr.message,
    'outgoing' AS direction,
    opr.status
FROM {{schema}}.organization_partner_request opr
JOIN {{schema}}.organization from_o ON from_o.uuid = opr.from_org_uuid
JOIN {{schema}}.organization to_o ON to_o.uuid = opr.to_org_uuid
WHERE opr.from_org_uuid = $1::uuid

UNION ALL

SELECT
    from_o.uuid AS org_uuid,
    from_o.name AS org_name,
    opr.created_at,
    opr.message,
    'incoming' AS direction,
    opr.status
FROM {{schema}}.organization_partner_request opr
JOIN {{schema}}.organization from_o ON from_o.uuid = opr.from_org_uuid
JOIN {{schema}}.organization to_o ON to_o.uuid = opr.to_org_uuid
WHERE opr.to_org_uuid = $1::uuid