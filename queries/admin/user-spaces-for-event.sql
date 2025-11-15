WITH direct_space_perms AS (
    SELECT
        o.id AS organizer_id,
        o.name AS organizer_name,
        v.id AS venue_id,
        v.name AS venue_name,
        LOWER(v.name) AS venue_sort_name,
        s.id AS space_id,
        s.name AS space_name,
        LOWER(s.name) AS space_sort_name,
        ur.add_event
    FROM {{schema}}.user_space_link usl
    JOIN {{schema}}.user_role ur ON usl.user_role_id = ur.id
    JOIN {{schema}}.space s ON usl.space_id = s.id
    JOIN {{schema}}.venue v ON s.venue_id = v.id
    JOIN {{schema}}.organizer o ON v.organizer_id = o.id
    WHERE usl.user_id = $1
),
venue_level_perms AS (
    SELECT
        o.id AS organizer_id,
        o.name AS organizer_name,
        v.id AS venue_id,
        v.name AS venue_name,
        LOWER(v.name) AS venue_sort_name,
        s.id AS space_id,
        s.name AS space_name,
        LOWER(s.name) AS space_sort_name,
        ur.add_event
    FROM {{schema}}.user_venue_link uvl
    JOIN {{schema}}.user_role ur ON uvl.user_role_id = ur.id
    JOIN {{schema}}.venue v ON uvl.venue_id = v.id
    JOIN {{schema}}.space s ON s.venue_id = v.id
    JOIN {{schema}}.organizer o ON v.organizer_id = o.id
    WHERE uvl.user_id = $1
),
organizer_level_perms AS (
    SELECT
        o.id AS organizer_id,
        o.name AS organizer_name,
        v.id AS venue_id,
        v.name AS venue_name,
        LOWER(v.name) AS venue_sort_name,
        s.id AS space_id,
        s.name AS space_name,
        LOWER(s.name) AS space_sort_name,
        ur.add_event
    FROM {{schema}}.user_organizer_link uol
    JOIN {{schema}}.user_role ur ON uol.user_role_id = ur.id
    JOIN {{schema}}.organizer o ON uol.organizer_id = o.id
    JOIN {{schema}}.venue v ON v.organizer_id = o.id
    JOIN {{schema}}.space s ON s.venue_id = v.id
    WHERE uol.user_id = $1
),
perms_union AS (
    SELECT * FROM direct_space_perms WHERE add_event = TRUE
    UNION ALL
    SELECT * FROM venue_level_perms WHERE add_event = TRUE
    UNION ALL
    SELECT * FROM organizer_level_perms WHERE add_event = TRUE
),
fixed_space AS (
    SELECT
        o.id AS organizer_id,
        o.name AS organizer_name,
        v.id AS venue_id,
        v.name AS venue_name,
        LOWER(v.name) AS venue_sort_name,
        s.id AS space_id,
        s.name AS space_name,
        LOWER(s.name) AS space_sort_name
    FROM {{schema}}.space s
    JOIN {{schema}}.venue v ON s.venue_id = v.id
    JOIN {{schema}}.organizer o ON v.organizer_id = o.id
    WHERE s.id = $2
)
SELECT DISTINCT ON (venue_sort_name, space_sort_name)
    organizer_id, organizer_name, venue_id, venue_name, space_id, space_name
FROM (
    SELECT organizer_id, organizer_name, venue_id, venue_name, space_id, space_name, venue_sort_name, space_sort_name
    FROM perms_union
    UNION ALL
    SELECT organizer_id, organizer_name, venue_id, venue_name, space_id, space_name, venue_sort_name, space_sort_name
    FROM fixed_space
) final
ORDER BY venue_sort_name, space_sort_name;