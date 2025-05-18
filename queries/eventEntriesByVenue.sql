SELECT
    e.id AS event_id,
    e.title AS event_title,
    uo.id AS organizer_id,
    uo.name AS organizer_name,
    v.id AS venue_id,
    v.name AS venue_name,
    COALESCE(s.id, es.id) AS space_id,
    COALESCE(s.name, es.name) AS space_name,
    ed.start AS start,
    ed.end AS end,
    ed.entry_time AS entry_time,
    ur.edit_event AS can_edit,
    ur.release_event AS can_release,
    ur.delete_event AS can_delete,
    ur.view_event_insights AS can_view_insights
FROM {{schema}}.event_date ed
         JOIN {{schema}}.event e ON ed.event_id = e.id
         JOIN {{schema}}.venue v ON COALESCE(ed.venue_id, e.venue_id) = v.id
         LEFT JOIN {{schema}}.space s ON ed.space_id = s.id
         LEFT JOIN {{schema}}.space es ON e.space_id = es.id
         LEFT JOIN {{schema}}.user_organizer_links uol ON uol.user_id = $2 AND uol.organizer_id = e.organizer_id
         LEFT JOIN {{schema}}.organizer uo ON uo.id = uol.organizer_id
         LEFT JOIN {{schema}}.user_role ur ON uol.user_role_id = ur.id
WHERE ed.start > NOW()
  AND COALESCE(ed.venue_id, e.venue_id) = $1
  AND uol.user_id = $2
ORDER BY ed.start ASC;
