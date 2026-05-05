SELECT v.uuid, v.name
FROM {{schema}}.venue v
WHERE v.org_uuid = $1
ORDER BY v.name