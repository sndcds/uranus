SELECT
    o.id AS organizer_id,
    o.name AS organizer_name,
    o.city AS organizer_city,
    o.country_code AS organizer_country,
    o.website_url AS organizer_web_url
FROM {{schema}}.user_event_organizer_links ueol
JOIN {{schema}}.organizer o ON o.id = ueol.organizer_id
WHERE user_id = $1;