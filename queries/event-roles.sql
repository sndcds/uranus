SELECT
    e.id AS event_id,
    e.title AS event_title,
    e.organizer_id AS event_organizer_id,
    eo.name AS event_organizer_name,
    s.id AS space_id,
    s.name AS space_name,
    s.venue_id,
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
FROM {{schema}}.user_event_links uel
    JOIN {{schema}}.event e ON e.id = uel.event_id
    JOIN {{schema}}.space s ON s.id = e.space_id
    JOIN {{schema}}.venue v ON v.id = s.venue_id
    JOIN {{schema}}.organizer o ON o.id = v.organizer_id
    JOIN {{schema}}.organizer eo ON eo.id = e.organizer_id
    JOIN {{schema}}.user_role ur ON ur.id = uel.user_role_id
WHERE uel.user_id = $1;