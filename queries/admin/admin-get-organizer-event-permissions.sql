SELECT
    (
        COALESCE(BOOL_OR(uolr.add_event), FALSE)
            OR COALESCE(BOOL_OR(uvlr.add_event), FALSE)
            OR COALESCE(BOOL_OR(uslr.add_event), FALSE)
        ) AS can_add_event
FROM {{schema}}."user" u
         LEFT JOIN {{schema}}.user_organizer_links uol
                   ON uol.user_id = u.id
                       AND uol.organizer_id = $2
         LEFT JOIN {{schema}}.user_role uolr ON uol.user_role_id = uolr.id

         LEFT JOIN {{schema}}.user_venue_links uvl
                   ON uvl.user_id = u.id
                       AND uvl.venue_id IN (
                           SELECT v.id FROM {{schema}}.venue v WHERE v.organizer_id = $2
                       )
         LEFT JOIN {{schema}}.user_role uvlr ON uvl.user_role_id = uvlr.id

         LEFT JOIN {{schema}}.user_space_links usl
                   ON usl.user_id = u.id
                       AND usl.space_id IN (
                           SELECT s.id
                           FROM {{schema}}.space s
                                    JOIN {{schema}}.venue v ON s.venue_id = v.id
                           WHERE v.organizer_id = $2
                       )
         LEFT JOIN {{schema}}.user_role uslr ON usl.user_role_id = uslr.id
WHERE u.id = $1