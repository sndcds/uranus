SELECT
    el.label,
    el.type,
    el.url
FROM {{schema}}.event_link el
WHERE el.event_uuid = $1