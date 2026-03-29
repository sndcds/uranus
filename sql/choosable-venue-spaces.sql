SELECT s.uuid, s.name
FROM {{schema}}.space s
WHERE s.venue_uuid = $1