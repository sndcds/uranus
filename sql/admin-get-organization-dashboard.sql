WITH upcoming_events AS (
    SELECT
        o.id AS organization_id,
        COUNT(ed.id) AS total_upcoming_events
    FROM {{schema}}.organization o
    LEFT JOIN {{schema}}.event e ON e.organization_id = o.id
    LEFT JOIN {{schema}}.event_date ed ON ed.event_id = e.id
    AND ed.start_date > CURRENT_DATE
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
        o.country AS organization_country,
        COALESCE(ae.total_upcoming_events, 0) AS total_upcoming_events,
        COALESCE(vc.venue_count, 0) AS venue_count,
        COALESCE(sc.space_count, 0) AS space_count,
        COALESCE(uol.permissions, 0) AS uer_permissions,
        main_logo_image_link.pluto_image_id AS main_logo_image_id
    FROM organization_access oa
    JOIN {{schema}}.organization o ON o.id = oa.organization_id
    LEFT JOIN upcoming_events ae ON ae.organization_id = o.id
    LEFT JOIN venue_counts vc ON vc.organization_id = o.id
    LEFT JOIN space_counts sc ON sc.organization_id = o.id
    LEFT JOIN {{schema}}.user_organization_link uol ON uol.organization_id = o.id AND uol.user_id = $1
    LEFT JOIN LATERAL (
        SELECT pil.pluto_image_id
        FROM {{schema}}.pluto_image_link pil
        WHERE pil.context = 'organization'
        AND pil.context_id = o.id
        AND pil.identifier = 'main_logo'
        ORDER BY pil.id
        LIMIT 1
    ) main_logo_image_link ON TRUE
)
SELECT
    organization_id,
    organization_name,
    organization_city,
    organization_country,
    total_upcoming_events,
    venue_count,
    space_count,
    uer_permissions,
    main_logo_image_id
FROM final_data
ORDER BY LOWER(organization_name)