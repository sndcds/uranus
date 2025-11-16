SELECT
    (
        COALESCE(BOOL_OR((uol.permissions & (1 << 24)) <> 0), FALSE)
            OR COALESCE(BOOL_OR((uvl.permissions & (1 << 24)) <> 0), FALSE)
            OR COALESCE(BOOL_OR((usl.permissions & (1 << 24)) <> 0), FALSE)
        ) AS can_add_event
FROM {{schema}}."user" u

-- Organizer links
LEFT JOIN {{schema}}.user_organizer_link uol
ON uol.user_id = u.id
    AND uol.organizer_id = $2

-- Venue links
    LEFT JOIN {{schema}}.user_venue_link uvl
    ON uvl.user_id = u.id
    AND uvl.venue_id IN (
    SELECT v.id
    FROM {{schema}}.venue v
    WHERE v.organizer_id = $2
    )

-- Space links
    LEFT JOIN {{schema}}.user_space_link usl
    ON usl.user_id = u.id
    AND usl.space_id IN (
    SELECT s.id
    FROM {{schema}}.space s
    JOIN {{schema}}.venue v ON s.venue_id = v.id
    WHERE v.organizer_id = $2
    )

WHERE u.id = $1