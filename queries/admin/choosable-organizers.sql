WITH user_org_access AS (
    SELECT DISTINCT
        o.id AS organizer_id,
        o.name AS organizer_name
    FROM {{schema}}.organizer o
    JOIN {{schema}}.user_organizer_links uol
    ON uol.organizer_id = o.id
    WHERE uol.user_id = $1
),
user_venue_access AS (
    SELECT DISTINCT
        o.id AS organizer_id,
        o.name AS organizer_name
    FROM {{schema}}.venue v
    JOIN {{schema}}.organizer o ON o.id = v.organizer_id
    JOIN {{schema}}.user_venue_links uvl ON uvl.venue_id = v.id
    WHERE uvl.user_id = $1
),
accessible_organizers AS (
    SELECT * FROM user_org_access
    UNION
    SELECT * FROM user_venue_access
)
SELECT
    organizer_id,
    organizer_name
FROM accessible_organizers
ORDER BY LOWER(organizer_name)