SELECT
    v.id AS venue_id,
    o.name AS org_name,
    v.name AS venue_name,
    ur.name AS role_name,
    ur.edit_organizer AS edit_org,
    ur.add_event,
    ur.edit_event,
    ur.delete_event,
    ur.release_event
FROM {{schema}}.user_organizer_links uol
JOIN {{schema}}.organizer o ON o.id = uol.organizer_id
JOIN {{schema}}.user_role ur ON ur.id = uol.user_role_id
JOIN {{schema}}.venue v ON v.organizer_id = o.id
WHERE uol.user_id = $1
    AND (ur.add_event OR ur.edit_event OR ur.delete_event OR ur.release_event)
ORDER BY org_name, venue_name