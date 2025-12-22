SELECT v.id, v.name
FROM {{schema}}.venue v
WHERE v.organization_id = $1
ORDER BY v.name