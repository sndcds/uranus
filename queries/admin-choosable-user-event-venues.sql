WITH direct_venues AS (
    SELECT v.id, v.name, v.city, v.country_code
    FROM {{schema}}.venue v
    JOIN {{schema}}.user_venue_link uvl
    ON uvl.venue_id = v.id
    WHERE uvl.user_id = $1
    AND (uvl.permissions & (1 << 11) <> 0)   -- bitmask check for can_choose_venue
),
indirect_venues AS (
    SELECT v.id, v.name, v.city, v.country_code
    FROM {{schema}}.venue v
    JOIN {{schema}}.organizer o
    ON o.id = v.organizer_id
    JOIN {{schema}}.user_organizer_link uol
    ON uol.organizer_id = o.id
    WHERE uol.user_id = $1
    AND (uol.permissions & (1 << 11) = <> 0)  -- bitmask check for can_choose_venue
)
SELECT DISTINCT id, name, city, country_code
FROM (
    SELECT * FROM direct_venues
    UNION
    SELECT * FROM indirect_venues
) AS all_venues
ORDER BY name