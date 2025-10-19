WITH user_org_access AS (
    SELECT DISTINCT o.id AS organizer_id
    FROM {{schema}}.organizer o
    JOIN {{schema}}.user_organizer_links uol ON uol.organizer_id = o.id
WHERE uol.user_id = $1
    ),
    user_venue_access AS (
SELECT DISTINCT v.organizer_id
FROM {{schema}}.venue v
    JOIN {{schema}}.user_venue_links uvl ON uvl.venue_id = v.id
WHERE uvl.user_id = $1
    ),
    accessible_organizers AS (
SELECT organizer_id FROM user_org_access
UNION
SELECT organizer_id FROM user_venue_access
    ),
    editable_venues AS (
SELECT v.id AS venue_id
FROM {{schema}}.venue v
    JOIN {{schema}}.user_venue_links uvl ON uvl.venue_id = v.id
    JOIN {{schema}}.user_role ur ON ur.id = uvl.user_role_id
WHERE uvl.user_id = $1
  AND (ur.edit_venue OR ur.delete_venue OR ur.add_space OR ur.edit_space OR ur.delete_space OR ur.add_event OR ur.edit_event OR ur.delete_event OR ur.release_event)
UNION
SELECT v.id AS venue_id
FROM {{schema}}.venue v
    JOIN {{schema}}.user_organizer_links uol ON uol.organizer_id = v.organizer_id
    JOIN {{schema}}.user_role ur ON ur.id = uol.user_role_id
WHERE uol.user_id = $1
  AND (ur.edit_venue OR ur.delete_venue OR ur.add_space OR ur.edit_space OR ur.delete_space OR ur.add_event OR ur.edit_event OR ur.delete_event OR ur.release_event)
    ),
    venue_permissions AS (
SELECT
    v.id AS venue_id,
    bool_or(ur.edit_venue) AS can_edit_venue,
    bool_or(ur.delete_venue) AS can_delete_venue,
    bool_or(ur.add_space) AS can_add_space,
    bool_or(ur.edit_space) AS can_edit_space,
    bool_or(ur.delete_space) AS can_delete_space,
    bool_or(ur.add_event) AS can_add_event,
    bool_or(ur.edit_event) AS can_edit_event,
    bool_or(ur.delete_event) AS can_delete_event,
    bool_or(ur.release_event) AS can_release_event
FROM {{schema}}.venue v
    LEFT JOIN {{schema}}.user_venue_links uvl ON uvl.venue_id = v.id AND uvl.user_id = $1
    LEFT JOIN {{schema}}.user_organizer_links uol ON uol.organizer_id = v.organizer_id AND uol.user_id = $1
    LEFT JOIN {{schema}}.user_role ur ON ur.id IN (uvl.user_role_id, uol.user_role_id)
GROUP BY v.id
    ),
    space_info AS (
SELECT
    s.id AS space_id,
    s.name AS space_name,
    s.venue_id,
    COUNT(DISTINCT ed.id) FILTER (WHERE ed.start > $3) AS upcoming_event_count
FROM {{schema}}.space s
    LEFT JOIN {{schema}}.event e ON e.space_id = s.id
    LEFT JOIN {{schema}}.event_date ed ON ed.event_id = e.id
GROUP BY s.id, s.name, s.venue_id
    ),
    venue_info AS (
SELECT
    v.id AS venue_id,
    v.name AS venue_name,
    v.organizer_id,
    CASE WHEN ev.venue_id IS NOT NULL THEN true ELSE false END AS can_edit,
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
GROUP BY v.id, v.name, v.organizer_id, ev.venue_id,
    vp.can_edit_venue, vp.can_delete_venue,
    vp.can_add_space, vp.can_edit_space, vp.can_delete_space,
    vp.can_add_event, vp.can_edit_event, vp.can_delete_event, vp.can_release_event
    ),
    organizer_permissions AS (
SELECT
    o.id AS organizer_id,
    bool_or(ur.edit_organizer) AS can_edit_organizer,
    bool_or(ur.delete_organizer) AS can_delete_organizer
FROM {{schema}}.organizer o
    LEFT JOIN {{schema}}.user_organizer_links uol ON uol.organizer_id = o.id AND uol.user_id = $1
    LEFT JOIN {{schema}}.user_role ur ON ur.id = uol.user_role_id
GROUP BY o.id
    ),
    organizer_info AS (
SELECT
    o.id AS organizer_id,
    o.name AS organizer_name,
    COALESCE(op.can_edit_organizer, false) AS can_edit_organizer,
    COALESCE(op.can_delete_organizer, false) AS can_delete_organizer,
    COALESCE(SUM(vi.upcoming_event_count), 0) AS total_upcoming_events,
    COALESCE(
    json_agg(
    json_build_object(
    'venue_id', vi.venue_id,
    'venue_name', vi.venue_name,
    'can_edit', vi.can_edit,
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
    ) FILTER (WHERE vi.venue_id IS NOT NULL),
    '[]'::json
    ) AS venues
FROM accessible_organizers ao
    JOIN {{schema}}.organizer o ON o.id = ao.organizer_id
    LEFT JOIN venue_info vi ON vi.organizer_id = o.id
    LEFT JOIN organizer_permissions op ON op.organizer_id = o.id
WHERE o.id = $2
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
               )
                   ORDER BY LOWER(oi.organizer_name)
       ) AS full_data
FROM organizer_info oi;