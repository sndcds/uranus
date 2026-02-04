SELECT
    eu.label,
    eu.type,
    eu.url
FROM {{schema}}.event_link eu
WHERE eu.event_id = $1