WITH organizer_perms AS (
    SELECT
        'organizer'::text AS entity_type,
        uol.organizer_id::integer AS entity_id,
        o.name::text AS entity_name,
        NULL::integer AS relation_id,
        r.*
    FROM {{schema}}.user_organizer_links uol
    JOIN {{schema}}.user_role r ON uol.user_role_id = r.id
    JOIN {{schema}}.organizer o ON uol.organizer_id = o.id
    WHERE uol.user_id = $1
),
venue_perms AS (
    SELECT
        'venue'::text AS entity_type,
        uvl.venue_id::integer AS entity_id,
        v.name::text AS entity_name,
        v.organizer_id::integer AS relation_id,
        r.*
    FROM {{schema}}.user_venue_links uvl
    JOIN {{schema}}.user_role r ON uvl.user_role_id = r.id
    JOIN {{schema}}.venue v ON uvl.venue_id = v.id
    WHERE uvl.user_id = $1
),
space_perms AS (
    SELECT
        'space'::text AS entity_type,
        usl.space_id::integer AS entity_id,
        s.name::text AS entity_name,
        s.venue_id::integer AS relation_id,
        r.*
    FROM {{schema}}.user_space_links usl
    JOIN {{schema}}.user_role r ON usl.user_role_id = r.id
    JOIN {{schema}}.space s ON usl.space_id = s.id
    WHERE usl.user_id = $1
),
event_perms AS (
    SELECT
        'event'::text AS entity_type,
        uel.event_id::integer AS entity_id,
        e.title::text AS entity_name,
        e.organizer_id AS relation_id,
        r.*
    FROM {{schema}}.user_event_links uel
    JOIN {{schema}}.user_role r ON uel.user_role_id = r.id
    JOIN {{schema}}.event e ON uel.event_id = e.id
    WHERE uel.user_id = $1
)
SELECT * FROM organizer_perms
UNION ALL
SELECT * FROM venue_perms
UNION ALL
SELECT * FROM space_perms
UNION ALL
SELECT * FROM event_perms;