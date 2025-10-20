SELECT s.id, s.name
FROM {{schema}}.space s
WHERE s.venue_id = $1