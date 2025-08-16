-- $1 user_id
-- $2 venue organizer_id
-- $3 earliest start
-- $4 latest start
WITH event_rights AS (
    SELECT
        e.id AS event_id,
        (
            BOOL_OR(ur_event.edit_event) OR
            BOOL_OR(ur_space.edit_event) OR
            BOOL_OR(ur_venue.edit_event) OR
            BOOL_OR(ur_org_event.edit_event) OR
            BOOL_OR(ur_org_venue.edit_event)
        ) AS can_edit,
        (
            BOOL_OR(ur_event.delete_event) OR
            BOOL_OR(ur_space.delete_event) OR
            BOOL_OR(ur_venue.delete_event) OR
            BOOL_OR(ur_org_event.delete_event) OR
            BOOL_OR(ur_org_venue.delete_event)
        ) AS can_delete,
        (
            BOOL_OR(ur_event.release_event) OR
            BOOL_OR(ur_space.release_event) OR
            BOOL_OR(ur_venue.release_event) OR
            BOOL_OR(ur_org_event.release_event) OR
            BOOL_OR(ur_org_venue.release_event)
        ) AS can_release
    FROM {{schema}}.event e

    -- Join event_date for filtering
    JOIN {{schema}}.event_date ed ON ed.event_id = e.id

    -- Event-level user link
    LEFT JOIN {{schema}}.user_event_links uel ON uel.event_id = e.id AND uel.user_id = $1
    LEFT JOIN {{schema}}.user_role ur_event ON ur_event.id = uel.user_role_id

    -- Space-level user link
    LEFT JOIN {{schema}}.space s ON s.id = e.space_id
    LEFT JOIN {{schema}}.user_space_links usl ON usl.space_id = s.id AND usl.user_id = $1
    LEFT JOIN {{schema}}.user_role ur_space ON ur_space.id = usl.user_role_id

    -- Venue-level user link
    LEFT JOIN {{schema}}.venue v ON v.id = s.venue_id
    LEFT JOIN {{schema}}.user_venue_links uvl ON uvl.venue_id = v.id AND uvl.user_id = $1
    LEFT JOIN {{schema}}.user_role ur_venue ON ur_venue.id = uvl.user_role_id

    -- Organizer-level user link (event.organizer_id)
    LEFT JOIN {{schema}}.user_organizer_links uol_event ON uol_event.organizer_id = e.organizer_id AND uol_event.user_id = $1
    LEFT JOIN {{schema}}.user_role ur_org_event ON ur_org_event.id = uol_event.user_role_id

    -- Organizer-level user link (venue.organizer_id)
    LEFT JOIN {{schema}}.user_organizer_links uol_venue ON uol_venue.organizer_id = v.organizer_id AND uol_venue.user_id = $1
    LEFT JOIN {{schema}}.user_role ur_org_venue ON ur_org_venue.id = uol_venue.user_role_id

    -- Filter by event date start
    WHERE v.organizer_id = $2 AND ed.start > $3 AND ed.start < $4

    GROUP BY e.id
)
SELECT
    o.id AS event_org_id,
    o.name AS event_org_name,
    v.id AS venue_id,
    v.name AS venue_name,
    vo.name AS venue_org_name,
    s.id AS space_id,
    s.name AS space_name,
    e.id AS event_id,
    e.title AS event_title,
    ed.id AS event_date_id,
    ed.start AS event_start,
    ed."end" AS event_end,
    COALESCE(er.can_edit, false) AS can_edit,
    COALESCE(er.can_delete, false) AS can_delete,
    COALESCE(er.can_release, false) AS can_release
FROM {{schema}}.event e
LEFT JOIN event_rights er ON er.event_id = e.id
LEFT JOIN {{schema}}.space s ON s.id = e.space_id
LEFT JOIN {{schema}}.venue v ON v.id = s.venue_id
LEFT JOIN {{schema}}.organizer o ON o.id = e.organizer_id
LEFT JOIN {{schema}}.organizer vo ON vo.id = v.organizer_id
JOIN {{schema}}.event_date ed ON ed.event_id = e.id
-- Same filter as above
WHERE v.organizer_id = $2 AND ed.start > $3 AND ed.start < $4
ORDER BY ed.start, e.title;