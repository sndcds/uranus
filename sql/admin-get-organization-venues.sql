WITH user_organization_access AS (
    SELECT DISTINCT o.uuid AS org_uuid
    FROM {{schema}}.organization o
    JOIN {{schema}}.user_organization_link uol ON uol.org_uuid = o.uuid
    WHERE uol.user_uuid = $1
),

user_venue_access AS (
    SELECT DISTINCT v.org_uuid
    FROM {{schema}}.venue v
    JOIN {{schema}}.user_venue_link uvl ON uvl.venue_uuid = v.uuid
    WHERE uvl.user_uuid = $1
),

accessible_organizations AS (
    SELECT org_uuid FROM user_organization_access

    UNION

    SELECT org_uuid FROM user_venue_access
),

editable_venues AS (
    SELECT v.uuid AS venue_uuid
    FROM {{schema}}.venue v
    JOIN {{schema}}.user_venue_link uvl ON uvl.venue_uuid = v.uuid
    WHERE uvl.user_uuid = $1
    AND (uvl.permissions &
        ((1<<9)|(1<<10)|(1<<16)|(1<<17)|(1<<18)|(1<<24)|(1<<25)|(1<<26)|(1<<27))
    ) <> 0

    UNION

    SELECT v.uuid AS venue_uuid
    FROM {{schema}}.venue v
    JOIN {{schema}}.user_organization_link uol ON uol.org_uuid = v.org_uuid
    WHERE uol.user_uuid = $1
    AND (uol.permissions &
        ((1<<9)|(1<<10)|(1<<16)|(1<<17)|(1<<18)|(1<<24)|(1<<25)|(1<<26)|(1<<27))
    ) <> 0
),

venue_permissions AS (
    SELECT
        v.uuid AS venue_uuid,
        bool_or((uvl.permissions & (1<<8)) <> 0 OR (uol.permissions & (1<<8)) <> 0) AS can_add_venue,
        bool_or((uvl.permissions & (1<<9)) <> 0 OR (uol.permissions & (1<<9)) <> 0) AS can_edit_venue,
        bool_or((uvl.permissions & (1<<10)) <> 0 OR (uol.permissions & (1<<10)) <> 0) AS can_delete_venue,
        bool_or((uvl.permissions & (1<<16)) <> 0 OR (uol.permissions & (1<<16)) <> 0) AS can_add_space,
        bool_or((uvl.permissions & (1<<17)) <> 0 OR (uol.permissions & (1<<17)) <> 0) AS can_edit_space,
        bool_or((uvl.permissions & (1<<18)) <> 0 OR (uol.permissions & (1<<18)) <> 0) AS can_delete_space,
        bool_or((uvl.permissions & (1<<24)) <> 0 OR (uol.permissions & (1<<24)) <> 0) AS can_add_event,
        bool_or((uvl.permissions & (1<<25)) <> 0 OR (uol.permissions & (1<<25)) <> 0) AS can_edit_event,
        bool_or((uvl.permissions & (1<<26)) <> 0 OR (uol.permissions & (1<<26)) <> 0) AS can_delete_event,
        bool_or((uvl.permissions & (1<<27)) <> 0 OR (uol.permissions & (1<<27)) <> 0) AS can_release_event
    FROM {{schema}}.venue v
    LEFT JOIN {{schema}}.user_venue_link uvl
    ON uvl.venue_uuid = v.uuid AND uvl.user_uuid = $1
    LEFT JOIN {{schema}}.user_organization_link uol
    ON uol.org_uuid = v.org_uuid AND uol.user_uuid = $1
    GROUP BY v.uuid
),

space_info AS (
    SELECT
        s.uuid AS space_uuid,
        s.name AS space_name,
        s.venue_uuid,
        COUNT(DISTINCT ed.uuid) FILTER (WHERE ed.start_date > $3) AS upcoming_event_count
    FROM {{schema}}.space s
    LEFT JOIN {{schema}}.event e ON e.space_uuid = s.uuid
    LEFT JOIN {{schema}}.event_date ed ON ed.event_uuid = e.uuid
    GROUP BY s.uuid, s.name, s.venue_uuid
),

venue_info AS (
    SELECT
        v.uuid AS venue_uuid,
        v.name AS venue_name,
        v.org_uuid,
        MAX(logos.main_logo_uuid::text) AS main_logo_uuid,
        MAX(logos.dark_theme_logo_uuid::text) AS dark_theme_logo_uuid,
        MAX(logos.light_theme_logo_uuid::text) AS light_theme_logo_uuid,
        COALESCE(vp.can_add_venue, false) AS can_add_venue,
        COALESCE(vp.can_edit_venue, false) AS can_edit_venue,
        COALESCE(vp.can_delete_venue, false) AS can_delete_venue,
        COALESCE(vp.can_add_space, false) AS can_add_space,
        COALESCE(vp.can_edit_space, false) AS can_edit_space,
        COALESCE(vp.can_delete_space, false) AS can_delete_space,
        COALESCE(vp.can_add_event, false) AS can_add_event,
        COALESCE(vp.can_edit_event, false) AS can_edit_event,
        COALESCE(vp.can_delete_event, false) AS can_delete_event,
        COALESCE(vp.can_release_event, false) AS can_release_event,
        COALESCE(SUM(s.upcoming_event_count), 0) AS upcoming_event_count,
        COALESCE(
        json_agg(
            json_build_object(
                'space_uuid', s.space_uuid,
                'space_name', s.space_name,
                'upcoming_event_count', s.upcoming_event_count
            )
        ) FILTER (WHERE s.space_uuid IS NOT NULL),
        '[]'::json
    ) AS spaces
    FROM {{schema}}.venue v
    LEFT JOIN space_info s ON s.venue_uuid = v.uuid
    LEFT JOIN editable_venues ev ON ev.venue_uuid = v.uuid
    LEFT JOIN venue_permissions vp ON vp.venue_uuid = v.uuid

    LEFT JOIN LATERAL (
        SELECT
            MAX(CASE WHEN pil.identifier = 'main_logo' THEN pil.pluto_image_uuid::text END) AS main_logo_uuid,
            MAX(CASE WHEN pil.identifier = 'dark_theme_logo' THEN pil.pluto_image_uuid::text END) AS dark_theme_logo_uuid,
            MAX(CASE WHEN pil.identifier = 'light_theme_logo' THEN pil.pluto_image_uuid::text END) AS light_theme_logo_uuid
        FROM {{schema}}.pluto_image_link pil
        WHERE pil.context = 'venue'
        AND pil.context_uuid = v.uuid
    ) logos ON TRUE

    GROUP BY
        v.uuid, v.name, v.org_uuid, ev.venue_uuid,
        vp.can_add_venue, vp.can_edit_venue, vp.can_delete_venue,
        vp.can_add_space, vp.can_edit_space, vp.can_delete_space,
        vp.can_add_event, vp.can_edit_event, vp.can_delete_event, vp.can_release_event
),

organization_permissions AS (
    SELECT
        o.uuid AS org_uuid,
        bool_or((uol.permissions & (1<<0)) <> 0) AS can_edit_org,
        bool_or((uol.permissions & (1<<1)) <> 0) AS can_delete_org,
        bool_or((uol.permissions & (1<<8)) <> 0) AS can_add_venue,
        bool_or((uol.permissions & (1<<16)) <> 0) AS can_add_space,
        bool_or((uol.permissions & (1<<24)) <> 0) AS can_add_event
    FROM {{schema}}.organization o
    LEFT JOIN {{schema}}.user_organization_link uol
    ON uol.org_uuid = o.uuid AND uol.user_uuid = $1
    GROUP BY o.uuid
),

organization_info AS (
    SELECT
        o.uuid AS org_uuid,
        o.name AS org_name,

        BOOL_OR(op.can_edit_org) AS can_edit_org,
        BOOL_OR(op.can_delete_org) AS can_delete_org,
        BOOL_OR(op.can_add_venue) AS can_add_venue,
        BOOL_OR(op.can_add_space) AS can_add_space,
        BOOL_OR(op.can_add_event) AS can_add_event,

        COALESCE(SUM(vi.upcoming_event_count), 0) AS total_upcoming_events,

        COALESCE(
            json_agg(
                json_build_object(
                    'venue_uuid', vi.venue_uuid,
                    'venue_name', vi.venue_name,
                    'venue_main_logo_uuid', main_logo_uuid,
                    'venue_dark_theme_logo_uuid', dark_theme_logo_uuid,
                    'venue_light_theme_logo_uuid', light_theme_logo_uuid,
                    'can_add_venue', vi.can_add_venue,
                    'can_edit_venue', vi.can_edit_venue,
                    'can_delete_venue', vi.can_delete_venue,
                    'can_add_space', vi.can_add_space,
                    'can_edit_space', vi.can_edit_space,
                    'can_delete_space', vi.can_delete_space,
                    'can_add_event', vi.can_add_event,
                    'can_edit_event', vi.can_edit_event,
                    'can_delete_event', vi.can_delete_event,
                    'can_release_event', vi.can_release_event,
                    'upcoming_event_count', vi.upcoming_event_count,
                    'spaces', vi.spaces
                )
                ORDER BY LOWER(vi.venue_name)
            ) FILTER (WHERE vi.venue_uuid IS NOT NULL),
            '[]'::json
        ) AS venues
    FROM accessible_organizations ao
    JOIN {{schema}}.organization o ON o.uuid = ao.org_uuid
    LEFT JOIN venue_info vi ON vi.org_uuid = o.uuid
    LEFT JOIN organization_permissions op ON op.org_uuid = o.uuid
    WHERE o.uuid = $2
    GROUP BY o.uuid, o.name
)

SELECT json_agg(
   json_build_object(
       'org_uuid', oi.org_uuid,
       'org_name', oi.org_name,
       'can_edit_org', oi.can_edit_org,
       'can_delete_org', oi.can_delete_org,
       'can_add_venue', oi.can_add_venue,
       'can_add_space', oi.can_add_space,
       'can_add_event', oi.can_add_event,
       'total_upcoming_events', oi.total_upcoming_events,
       'venues', oi.venues
   )
   ORDER BY LOWER(oi.org_name)
) AS full_data
FROM organization_info oi
