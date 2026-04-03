WITH upcoming_events AS (
    SELECT
        o.uuid AS org_uuid,
        COUNT(ed.uuid) AS total_upcoming_events
    FROM {{schema}}.organization o
    LEFT JOIN {{schema}}.event e ON e.org_uuid = o.uuid
    LEFT JOIN {{schema}}.event_date ed ON ed.event_uuid = e.uuid
    AND ed.start_date > CURRENT_DATE
    GROUP BY o.uuid
),
organization_access AS (
    SELECT DISTINCT org_uuid
    FROM {{schema}}.user_organization_link
    WHERE user_uuid = $1

    UNION

    SELECT v.org_uuid
    FROM {{schema}}.venue v
    JOIN {{schema}}.user_venue_link uvl ON uvl.venue_uuid = v.uuid
    WHERE uvl.user_uuid = $1
),
venue_counts AS (
    SELECT o.uuid AS org_uuid,
    COUNT(v.uuid) AS venue_count
    FROM {{schema}}.organization o
    LEFT JOIN {{schema}}.venue v ON v.org_uuid = o.uuid
    GROUP BY o.uuid
),
space_counts AS (
    SELECT o.uuid AS org_uuid,
    COUNT(s.uuid) AS space_count
    FROM {{schema}}.organization o
    LEFT JOIN {{schema}}.venue v ON v.org_uuid = o.uuid
    LEFT JOIN {{schema}}.space s ON s.venue_uuid = v.uuid
    GROUP BY o.uuid
),
final_data AS (
    SELECT
        o.uuid AS uuid,
        o.name AS name,
        o.city AS city,
        o.country AS country,
        COALESCE(ae.total_upcoming_events, 0) AS total_upcoming_events,
        COALESCE(vc.venue_count, 0) AS venue_count,
        COALESCE(sc.space_count, 0) AS space_count,
        COALESCE(uol.permissions, 0) AS uer_permissions,
        main_logo_link.pluto_image_uuid AS main_logo_uuid,
        dark_theme_logo_link.pluto_image_uuid AS dark_theme_logo_uuid,
        light_theme_logo_link.pluto_image_uuid AS light_theme_logo_uuid
    FROM organization_access oa
    JOIN {{schema}}.organization o ON o.uuid = oa.org_uuid
    LEFT JOIN upcoming_events ae ON ae.org_uuid = o.uuid
    LEFT JOIN venue_counts vc ON vc.org_uuid = o.uuid
    LEFT JOIN space_counts sc ON sc.org_uuid = o.uuid
    LEFT JOIN {{schema}}.user_organization_link uol ON uol.org_uuid = o.uuid AND uol.user_uuid = $1
    LEFT JOIN LATERAL (
        SELECT pil.pluto_image_uuid FROM {{schema}}.pluto_image_link pil
        WHERE pil.context = 'organization' AND pil.context_uuid = o.uuid AND pil.identifier = 'main_logo' LIMIT 1
    ) main_logo_link ON TRUE
    LEFT JOIN LATERAL (
        SELECT pil.pluto_image_uuid FROM {{schema}}.pluto_image_link pil
        WHERE pil.context = 'organization' AND pil.context_uuid = o.uuid AND pil.identifier = 'dark_theme_logo' LIMIT 1
    ) dark_theme_logo_link ON TRUE
    LEFT JOIN LATERAL (
        SELECT pil.pluto_image_uuid FROM {{schema}}.pluto_image_link pil
        WHERE pil.context = 'organization' AND pil.context_uuid = o.uuid AND pil.identifier = 'light_theme_logo' LIMIT 1
    ) light_theme_logo_link ON TRUE
)
SELECT
    uuid,
    name,
    city,
    country,
    total_upcoming_events,
    venue_count,
    space_count,
    uer_permissions,
    main_logo_uuid,
    dark_theme_logo_uuid,
    light_theme_logo_uuid
FROM final_data
ORDER BY LOWER(name)