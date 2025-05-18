WITH user_rights AS (
    SELECT
        uol.organizer_id,
        BOOL_OR(ur.edit_organizer) AS can_edit_organizer,
        BOOL_OR(ur.delete_organizer) AS can_delete_organizer,
        BOOL_OR(ur.add_venue) AS can_add_venue,
        BOOL_OR(ur.edit_venue) AS can_edit_venue,
        BOOL_OR(ur.delete_venue) AS can_delete_venue,
        BOOL_OR(ur.add_space) AS can_add_space,
        BOOL_OR(ur.edit_space) AS can_edit_space,
        BOOL_OR(ur.delete_space) AS can_delete_space,
        BOOL_OR(ur.add_event) AS can_add_event,
        BOOL_OR(ur.edit_event) AS can_edit_event,
        BOOL_OR(ur.delete_event) AS can_delete_event,
        BOOL_OR(ur.release_event) AS can_release_event,
        BOOL_OR(ur.view_event_insights) AS can_view_event_insights
    FROM {{schema}}.user_organizer_links uol
        JOIN {{schema}}.user_role ur ON uol.user_role_id = ur.id
    WHERE uol.user_id = $1
    GROUP BY uol.organizer_id
),
     venue_counts AS (
         SELECT organizer_id, COUNT(*) AS venue_count
         FROM {{schema}}.venue
         WHERE organizer_id IS NOT NULL
         GROUP BY organizer_id
     ),
     space_counts AS (
         SELECT v.organizer_id, COUNT(s.id) AS space_count
         FROM {{schema}}.space s
            JOIN {{schema}}.venue v ON s.venue_id = v.id
         WHERE v.organizer_id IS NOT NULL
         GROUP BY v.organizer_id
     ),
     event_counts AS (
         SELECT e.organizer_id, COUNT(ed.id) AS upcoming_event_count
         FROM {{schema}}.event e
            JOIN {{schema}}.event_date ed ON e.id = ed.event_id
         WHERE ed.start >= NOW() -- Only count future events
         GROUP BY e.organizer_id
     )
SELECT
    o.id AS organizer_id,
    o.name AS organizer_name,
    o.street AS organizer_street,
    o.house_number AS organizer_house_number,
    o.postal_code AS organizer_postal_code,
    o.city AS organizer_city,
    o.website_url AS organizer_website_url,
    ur.can_edit_organizer,
    ur.can_delete_organizer,
    ur.can_add_venue,
    ur.can_edit_venue,
    ur.can_delete_venue,
    ur.can_add_space,
    ur.can_edit_space,
    ur.can_delete_space,
    ur.can_add_event,
    ur.can_edit_event,
    ur.can_delete_event,
    ur.can_release_event,
    ur.can_view_event_insights,
    COALESCE(vc.venue_count, 0) AS venue_count,
    COALESCE(sc.space_count, 0) AS space_count,
    COALESCE(ec.upcoming_event_count, 0) AS upcoming_event_count
FROM {{schema}}.organizer o
    JOIN user_rights ur ON o.id = ur.organizer_id
    LEFT JOIN venue_counts vc ON o.id = vc.organizer_id
    LEFT JOIN space_counts sc ON o.id = sc.organizer_id
    LEFT JOIN event_counts ec ON o.id = ec.organizer_id
ORDER BY LOWER(o.name);