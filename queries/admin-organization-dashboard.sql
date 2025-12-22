WITH upcoming_events AS (
    SELECT
        o.id AS organization_id,
        COUNT(DISTINCT ed.id) AS total_upcoming_events
    FROM {{schema}}.organization o
    LEFT JOIN {{schema}}.venue v ON v.organization_id = o.id
    LEFT JOIN {{schema}}.event e ON (e.venue_id = v.id OR e.organization_id = o.id)
    LEFT JOIN {{schema}}.event_date ed ON ed.event_id = e.id
    WHERE ed.start_date > CURRENT_DATE
    GROUP BY o.id
),
organization_access AS (
    SELECT DISTINCT organization_id
    FROM {{schema}}.user_organization_link
    WHERE user_id = $1

    UNION

    SELECT v.organization_id
    FROM {{schema}}.venue v
    JOIN {{schema}}.user_venue_link uvl ON uvl.venue_id = v.id
    WHERE uvl.user_id = $1
),
venue_counts AS (
    SELECT o.id AS organization_id,
    COUNT(v.id) AS venue_count
    FROM {{schema}}.organization o
    LEFT JOIN {{schema}}.venue v ON v.organization_id = o.id
    GROUP BY o.id
),
space_counts AS (
    SELECT o.id AS organization_id,
    COUNT(s.id) AS space_count
    FROM {{schema}}.organization o
    LEFT JOIN {{schema}}.venue v ON v.organization_id = o.id
    LEFT JOIN {{schema}}.space s ON s.venue_id = v.id
    GROUP BY o.id
),
final_data AS (
    SELECT
        o.id AS organization_id,
        o.name AS organization_name,
        o.city AS organization_city,
        o.country_code AS organization_country_code,
        COALESCE(ae.total_upcoming_events, 0) AS total_upcoming_events,
        COALESCE(vc.venue_count, 0) AS venue_count,
        COALESCE(sc.space_count, 0) AS space_count,
        COALESCE(uol.permissions, 0) AS uer_permissions
    FROM organization_access oa
    JOIN {{schema}}.organization o ON o.id = oa.organization_id
    LEFT JOIN upcoming_events ae ON ae.organization_id = o.id
    LEFT JOIN venue_counts vc ON vc.organization_id = o.id
    LEFT JOIN space_counts sc ON sc.organization_id = o.id
    LEFT JOIN {{schema}}.user_organization_link uol ON uol.organization_id = o.id AND uol.user_id = $1
)
SELECT
    organization_id,
    organization_name,
    organization_city,
    organization_country_code,
    total_upcoming_events,
    venue_count,
    space_count,
    uer_permissions
FROM final_data
ORDER BY LOWER(organization_name)