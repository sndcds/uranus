WITH direct_space_perms AS (
    SELECT
        o.uuid AS org_id,
        o.name AS org_name,
        v.uuid AS venue_uuid,
        v.name AS venue_name,
        LOWER(v.name) AS venue_sort_name,
        s.uuid AS space_uuid,
        s.name AS space_name,
        LOWER(s.name) AS space_sort_name,
        ur.add_event
    FROM {{schema}}.user_space_link usl
    JOIN {{schema}}.user_role ur ON usl.user_role_id = ur.id
    JOIN {{schema}}.space s ON usl.space_uuid = s.uuid
    JOIN {{schema}}.venue v ON s.venue_uuid = v.uuid
    JOIN {{schema}}.organization o ON v.org_uuid = o.uuid
    WHERE usl.user_uuid = $1
),
venue_level_perms AS (
    SELECT
        o.uuid AS org_uuid,
        o.name AS org_name,
        v.uuid AS venue_uuid,
        v.name AS venue_name,
        LOWER(v.name) AS venue_sort_name,
        s.uuid AS space_uuid,
        s.name AS space_name,
        LOWER(s.name) AS space_sort_name,
        ur.add_event
    FROM {{schema}}.user_venue_link uvl
    JOIN {{schema}}.user_role ur ON uvl.user_role_id = ur.id
    JOIN {{schema}}.venue v ON uvl.venue_uuid = v.uuid
    JOIN {{schema}}.space s ON s.venue_uuid = v.uuid
    JOIN {{schema}}.organization o ON v.org_uuid = o.uuid
    WHERE uvl.user_uuid = $1
),
organization_level_perms AS (
    SELECT
        o.uuid AS org_uuid,
        o.name AS org_name,
        v.uuid AS venue_uuid,
        v.name AS venue_name,
        LOWER(v.name) AS venue_sort_name,
        s.uuid AS space_uuid,
        s.name AS space_name,
        LOWER(s.name) AS space_sort_name,
        ur.add_event
    FROM {{schema}}.user_organization_link uol
    JOIN {{schema}}.user_role ur ON uol.user_role_id = ur.id
    JOIN {{schema}}.organization o ON uol.org_uuid = o.uuid
    JOIN {{schema}}.venue v ON v.org_uuid = o.uuid
    JOIN {{schema}}.space s ON s.venue_uuid = v.uuid
    WHERE uol.user_uuid = $1
),
perms_union AS (
    SELECT * FROM direct_space_perms WHERE add_event = TRUE
    UNION ALL
    SELECT * FROM venue_level_perms WHERE add_event = TRUE
    UNION ALL
    SELECT * FROM organization_level_perms WHERE add_event = TRUE
),
fixed_space AS (
    SELECT
        o.uuid AS org_uuid,
        o.name AS org_name,
        v.uuid AS venue_uuid,
        v.name AS venue_name,
        LOWER(v.name) AS venue_sort_name,
        s.uuid AS space_uuid,
        s.name AS space_name,
        LOWER(s.name) AS space_sort_name
    FROM {{schema}}.space s
    JOIN {{schema}}.venue v ON s.venue_uuid = v.uuid
    JOIN {{schema}}.organization o ON v.org_uuid = o.uuid
    WHERE s.uuid = $2
)
SELECT DISTINCT ON (venue_sort_name, space_sort_name)
    org_uuid, org_name, venue_uuid, venue_name, space_uuid, space_name
FROM (
    SELECT
        org_uuid,
        org_name,
        venue_uuid,
        venue_name,
        space_uuid,
        space_name,
        venue_sort_name,
        space_sort_name
    FROM perms_union
    UNION ALL
    SELECT
        org_uuid,
        org_name,
        venue_uuid,
        venue_name,
        space_uuid,
        space_name,
        venue_sort_name,
        space_sort_name
    FROM fixed_space
) final
ORDER BY venue_sort_name, space_sort_name