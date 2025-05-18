SELECT
    e.id AS event_id,
    e.title AS event_title,
    o.id AS organizer_id,
    o.name AS organizer_name,
    v.id AS venue_id,
    v.name AS venue_name,
    COALESCE(s.id, es.id) AS space_id,
    COALESCE(s.name, es.name) AS space_name,
    ed.start AS start,
    ed.end AS end,
    TO_CHAR(ed.entry_time, 'HH24:MI') AS entry_time,
    ur.edit_event AS can_edit,
    ur.release_event AS can_release,
    ur.delete_event AS can_delete,
    ur.view_event_insights AS can_view_insights,
    EXISTS (
        SELECT 1
        FROM {{schema}}event_image_links eli
        WHERE eli.event_id = e.id
          AND eli.main_image = TRUE
    ) AS has_main_image,
    img.source_name AS img_source_name
FROM {{schema}}.event_date ed
         JOIN {{schema}}.event e ON ed.event_id = e.id
         LEFT JOIN {{schema}}.space s ON ed.space_id = s.id
         LEFT JOIN {{schema}}.space es ON e.space_id = es.id
         JOIN {{schema}}.venue v ON COALESCE(s.venue_id, es.venue_id) = v.id
         LEFT JOIN {{schema}}.user_organizer_links uol ON uol.user_id = $2
    AND uol.organizer_id = e.organizer_id
         LEFT JOIN {{schema}}.user_role ur ON uol.user_role_id = ur.id
         JOIN {{schema}}.organizer o ON e.organizer_id = o.id
         LEFT JOIN {{schema}}.event_image_links eli ON eli.event_id = e.id
    AND eli.main_image = TRUE
         LEFT JOIN {{schema}}.image img ON eli.image_id = img.id
WHERE ed.start > NOW()
  AND o.id = $1
ORDER BY ed.start ASC;