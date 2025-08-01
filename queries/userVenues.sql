SELECT id, name, postal_code, city, country_code, organizer_id
FROM (
    SELECT v.*
        FROM {{schema}}.user_venue_links uvl
    JOIN {{schema}}.user_role ur ON uvl.user_role_id = ur.id
    JOIN {{schema}}.venue v ON v.id = uvl.venue_id
    WHERE uvl.user_id = $1
        AND ur.add_event = TRUE

    UNION

    SELECT v.*
         FROM {{schema}}.user_organizer_links uol
    JOIN {{schema}}.user_role ur ON uol.user_role_id = ur.id
    JOIN {{schema}}.venue v ON v.organizer_id = uol.organizer_id
    WHERE uol.user_id = $1
        AND ur.edit_event = TRUE
    ) AS combined_venues
ORDER BY name
