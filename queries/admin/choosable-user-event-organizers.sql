SELECT *
FROM (
    SELECT
        o.id AS id,
        o.name AS name,
        o.city AS city,
        o.country_code AS country_code
    FROM {{schema}}.user_event_organizer_links ueol
    JOIN {{schema}}.organizer o ON o.id = ueol.event_organizer_id
    WHERE ueol.user_id = $1
    AND ueol.organizer_id = $2

    UNION

    SELECT
        o.id AS id,
        o.name AS name,
        o.city AS city,
        o.country_code AS country_code
    FROM {{schema}}.organizer o
    WHERE o.id = $2
) combined
ORDER BY LOWER(name)