SELECT
    eu.id,
    eu.url_type,
    eu.url,
    eu.title
FROM {{schema}}.event_weblink eu
WHERE eu.event_id = $1