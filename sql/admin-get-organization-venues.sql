WITH event_dates AS (
    SELECT *
    FROM {{schema}}.event_date
    WHERE start_date >= $3
),

resolved_events AS (
    SELECT
        ed.uuid AS event_date_uuid,
        COALESCE(ed.space_uuid, e.space_uuid) AS space_uuid
    FROM event_dates ed
    LEFT JOIN {{schema}}.event e ON e.uuid = ed.event_uuid
),

space_event_counts AS (
    SELECT
        s.uuid AS space_uuid,
        s.venue_uuid,
        s.name AS space_name,
        COUNT(re.event_date_uuid) AS event_count
    FROM {{schema}}.space s
    LEFT JOIN resolved_events re ON re.space_uuid = s.uuid
    GROUP BY s.uuid, s.venue_uuid, s.name
),

venue_spaces AS (
    SELECT
        sec.venue_uuid,
        jsonb_agg(
            jsonb_build_object (
                'space_uuid', sec.space_uuid,
                'space_name', sec.space_name,
                'event_count', sec.event_count
            )
            ORDER BY sec.space_name
        ) AS spaces
    FROM space_event_counts sec
    GROUP BY sec.venue_uuid
),

venue_images AS (
    SELECT
        context_uuid AS venue_uuid,
        MAX(CASE WHEN identifier = 'main_logo' THEN pluto_image_uuid::text END) AS main_logo_uuid,
        MAX(CASE WHEN identifier = 'dark_theme_logo' THEN pluto_image_uuid::text END) AS dark_theme_logo_uuid,
        MAX(CASE WHEN identifier = 'light_theme_logo' THEN pluto_image_uuid::text END) AS light_theme_logo_uuid
    FROM {{schema}}.pluto_image_link
    WHERE context = 'venue'
    GROUP BY context_uuid
),

permissions AS (
    SELECT
        v.uuid AS venue_uuid,
        COALESCE(
            MAX(uol.permissions),
            0
        )
        |
        COALESCE(
            MAX(uvl.permissions),
            0
        ) AS permissions
    FROM {{schema}}.venue v
    JOIN {{schema}}.organization o ON o.uuid = v.org_uuid

    LEFT JOIN {{schema}}.user_organization_link uol
    ON uol.org_uuid = o.uuid
    AND uol.user_uuid = $2::uuid

    LEFT JOIN {{schema}}.user_venue_link uvl
    ON uvl.venue_uuid = v.uuid
    AND uvl.user_uuid = $2::uuid

    GROUP BY v.uuid
)

SELECT
    v.uuid AS venue_uuid,
    v.name AS venue_name,

    vi.main_logo_uuid,
    vi.dark_theme_logo_uuid,
    vi.light_theme_logo_uuid,

    COALESCE(
        vs.spaces,
        '[]'::jsonb
    ) AS spaces,

    p.permissions

FROM {{schema}}.organization o
LEFT JOIN {{schema}}.venue v
ON v.org_uuid = o.uuid

LEFT JOIN venue_images vi
ON vi.venue_uuid = v.uuid

LEFT JOIN venue_spaces vs
ON vs.venue_uuid = v.uuid

LEFT JOIN permissions p
ON p.venue_uuid = v.uuid

WHERE o.uuid = $1::uuid

GROUP BY
    v.uuid,
    v.name,
    vi.main_logo_uuid,
    vi.dark_theme_logo_uuid,
    vi.light_theme_logo_uuid,
    vs.spaces,
    p.permissions