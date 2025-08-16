SELECT
    v.id AS venue_id,
    v.name AS venue_name,
    v.organizer_id,
    o.name AS organizer_name,
    ur.id AS user_role_id,
    ur.name AS role_name,
    ur.edit_organizer,
    ur.delete_organizer,
    ur.add_venue,
    ur.edit_venue,
    ur.delete_venue,
    ur.add_space,
    ur.edit_space,
    ur.delete_space,
    ur.add_event,
    ur.edit_event,
    ur.delete_event,
    ur.release_event,
    ur.view_event_insights
FROM {{schema}}.user_venue_links uvl
    JOIN {{schema}}.venue v ON v.id = uvl.venue_id
    JOIN {{schema}}.organizer o ON o.id = v.organizer_id
    JOIN {{schema}}.user_role ur ON ur.id = uvl.user_role_id
WHERE uvl.user_id = $1;