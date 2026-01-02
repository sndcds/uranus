SELECT *
FROM (
    SELECT
        o.id AS id,
        o.name AS name,
        o.city AS city,
        o.country_code AS country_code
    FROM {{schema}}.user_event_organization_link ueol
    JOIN {{schema}}.organization o ON o.id = ueol.event_organization_id
    WHERE ueol.user_id = $1
    AND ueol.organization_id = $2

    UNION

    SELECT
        o.id AS id,
        o.name AS name,
        o.city AS city,
        o.country_code AS country_code
    FROM {{schema}}.organization o
    WHERE o.id = $2
) combined
ORDER BY LOWER(name)