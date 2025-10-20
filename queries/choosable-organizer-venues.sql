SELECT v.id, v.name
FROM {{schema}}.venue v
WHERE v.organizer_id = $1