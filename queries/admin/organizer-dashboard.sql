-- Define masks for permissions (as per your bit assignments)
WITH masks AS (
    SELECT
    -- Organizer permissions: EditOrganizer | DeleteOrganizer
    ((1<<0) | (1<<1)) AS organizer_edit_mask,
     -- Venue permissions: AddVenue, EditVenue, DeleteVenue, ChooseVenue
    ((1<<8) | (1<<9) | (1<<10) | (1<<11)) AS venue_mask,
     -- Space permissions: AddSpace, EditSpace, DeleteSpace
    ((1<<16) | (1<<17) | (1<<18)) AS space_mask,
     -- Event permissions: AddEvent, EditEvent, DeleteEvent, ReleaseEvent, ViewEvenInsights
    ((1<<24) | (1<<25) | (1<<26) | (1<<27) | (1<<28)) AS event_mask
),

-- Organizers accessible via user links
user_org_access AS (
    SELECT DISTINCT o.id AS organizer_id
    FROM {{schema}}.organizer o
    JOIN {{schema}}.user_organizer_link uol ON uol.organizer_id = o.id
    WHERE uol.user_id = $1
),

-- Organizers accessible via venue links
user_venue_access AS (
    SELECT DISTINCT v.organizer_id
    FROM {{schema}}.venue v
    JOIN {{schema}}.user_venue_link uvl ON uvl.venue_id = v.id
    WHERE uvl.user_id = $1
),

accessible_organizers AS (
    SELECT organizer_id
    FROM user_org_access

    UNION

    SELECT organizer_id
    FROM user_venue_access
),

-- Venues editable by the user (using bitmask)
editable_venues AS (
    SELECT v.id AS venue_id
    FROM {{schema}}.venue v
    JOIN {{schema}}.user_venue_link uvl ON uvl.venue_id = v.id
    JOIN masks m ON TRUE
    WHERE uvl.user_id = $1 AND (uvl.permissions & (m.venue_mask | m.space_mask | m.event_mask)) != 0

    UNION

    SELECT v.id AS venue_id
    FROM {{schema}}.venue v
    JOIN {{schema}}.user_organizer_link uol ON uol.organizer_id = v.organizer_id
    JOIN masks m ON TRUE
    WHERE uol.user_id = $1
    AND (uol.permissions & (m.venue_mask | m.space_mask | m.event_mask)) != 0
),

-- Aggregate venue permissions
venue_permissions AS (
    SELECT
        v.id AS venue_id,
        (COALESCE(uvl.permissions,0) & (1<<9)) != 0 AS can_edit_venue,
        (COALESCE(uvl.permissions,0) & (1<<10)) != 0 AS can_delete_venue,
        (COALESCE(uvl.permissions,0) & ((1<<16)|(1<<17)|(1<<18))) != 0 AS can_manage_spaces,
        (COALESCE(uvl.permissions,0) & ((1<<24)|(1<<25)|(1<<26)|(1<<27))) != 0 AS can_manage_events
    FROM {{schema}}.venue v
    LEFT JOIN {{schema}}.user_venue_link uvl
    ON uvl.venue_id = v.id AND uvl.user_id = $1
),

-- Space-level event counts
space_info AS (
    SELECT
        s.id AS space_id,
        s.name AS space_name,
        s.venue_id,
        COUNT(DISTINCT ed.id) FILTER (WHERE ed.start > $2) AS upcoming_event_count
    FROM {{schema}}.space s
    LEFT JOIN {{schema}}.event e ON e.space_id = s.id
    LEFT JOIN {{schema}}.event_date ed ON ed.event_id = e.id
    GROUP BY s.id, s.name, s.venue_id
),

-- Venue-level event counts (events without space)
venue_event_counts AS (
    SELECT
        v.id AS venue_id,
        COUNT(DISTINCT ed.id) FILTER (WHERE ed.start > $2) AS upcoming_event_count
    FROM {{schema}}.venue v
    LEFT JOIN {{schema}}.event e ON e.venue_id = v.id AND e.space_id IS NULL
    LEFT JOIN {{schema}}.event_date ed ON ed.event_id = e.id
    GROUP BY v.id
),

-- Venue info with aggregated permissions and event counts
venue_info AS (
    SELECT
        v.id AS venue_id,
        v.name AS venue_name,
        v.organizer_id,
        CASE WHEN ev.venue_id IS NOT NULL THEN TRUE ELSE FALSE END AS can_edit,
        COALESCE(vp.can_edit_venue, FALSE) AS can_edit_venue,
        COALESCE(vp.can_delete_venue, FALSE) AS can_delete_venue,
        COALESCE(vp.can_manage_spaces, FALSE) AS can_manage_spaces,
        COALESCE(vp.can_manage_events, FALSE) AS can_manage_events,
        COALESCE(SUM(s.upcoming_event_count),0) + COALESCE(vec.upcoming_event_count,0) AS upcoming_event_count,
        COALESCE(
            json_agg(
                json_build_object(
                    'space_id', s.space_id,
                    'space_name', s.space_name,
                    'upcoming_event_count', s.upcoming_event_count
                )
            ) FILTER (WHERE s.space_id IS NOT NULL),
            '[]'::json
        ) AS spaces
    FROM {{schema}}.venue v
    LEFT JOIN space_info s ON s.venue_id = v.id
    LEFT JOIN editable_venues ev ON ev.venue_id = v.id
    LEFT JOIN venue_permissions vp ON vp.venue_id = v.id
    LEFT JOIN venue_event_counts vec ON vec.venue_id = v.id
    GROUP BY
        v.id, v.name, v.organizer_id, ev.venue_id,
        vp.can_edit_venue, vp.can_delete_venue,
        vp.can_manage_spaces, vp.can_manage_events,
        vec.upcoming_event_count
),

-- Organizer-level permissions using bitmask
organizer_permissions AS (
    SELECT
        o.id AS organizer_id,
        (COALESCE(uol.permissions,0) & ((1<<0)|(1<<1))) != 0 AS can_edit_organizer,
        (COALESCE(uol.permissions,0) & ((1<<0)|(1<<1))) != 0 AS can_delete_organizer
    FROM {{schema}}.organizer o
    LEFT JOIN {{schema}}.user_organizer_link uol
    ON uol.organizer_id = o.id AND uol.user_id = $1
),

-- Final organizer info
organizer_info AS (
    SELECT
        o.id AS organizer_id,
        o.name AS organizer_name,
        COALESCE(op.can_edit_organizer,FALSE) AS can_edit_organizer,
        COALESCE(op.can_delete_organizer,FALSE) AS can_delete_organizer,
        COALESCE(SUM(vi.upcoming_event_count),0) AS total_upcoming_events,
        COALESCE(
            json_agg(
                json_build_object(
                'venue_id', vi.venue_id,
                'venue_name', vi.venue_name,
                'can_edit', vi.can_edit,
                'can_edit_venue', vi.can_edit_venue,
                'can_delete_venue', vi.can_delete_venue,
                'can_manage_spaces', vi.can_manage_spaces,
                'can_manage_events', vi.can_manage_events,
                'upcoming_event_count', vi.upcoming_event_count,
                'spaces', vi.spaces
                )
            ) FILTER (WHERE vi.venue_id IS NOT NULL),
            '[]'::json
        ) AS venues
    FROM accessible_organizers ao
    JOIN {{schema}}.organizer o ON o.id = ao.organizer_id
    LEFT JOIN venue_info vi ON vi.organizer_id = o.id
    LEFT JOIN organizer_permissions op ON op.organizer_id = o.id
    GROUP BY o.id, o.name, op.can_edit_organizer, op.can_delete_organizer
)

SELECT json_agg(
    json_build_object(
       'organizer_id', oi.organizer_id,
       'organizer_name', oi.organizer_name,
       'can_edit_organizer', oi.can_edit_organizer,
       'can_delete_organizer', oi.can_delete_organizer,
       'total_upcoming_events', oi.total_upcoming_events,
       'venues', oi.venues
    ) ORDER BY LOWER(oi.organizer_name)
) AS full_data
FROM organizer_info oi