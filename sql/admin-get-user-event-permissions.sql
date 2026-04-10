SELECT DISTINCT
(COALESCE(uol.permissions, 0) | COALESCE(uvl.permissions, 0)) AS permissions
FROM {{schema}}.event_date_projection edp
JOIN {{schema}}.user u ON u.uuid = $1::uuid
JOIN {{schema}}.event_projection ep ON ep.event_uuid = edp.event_uuid
LEFT JOIN {{schema}}.user_venue_link uvl ON uvl.user_uuid = u.uuid
LEFT JOIN {{schema}}.user_organization_link uol ON uol.user_uuid = u.uuid
WHERE edp.event_date_uuid = $2::uuid