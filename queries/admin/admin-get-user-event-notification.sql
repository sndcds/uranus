SELECT
    e.id AS event_id,
    e.title,
    e.release_date,
    (e.release_date - CURRENT_DATE) AS days_until_release,
    e.organizer_id,
    o.name AS organizer_name,
    ed_min.first_event_date::date AS earliest_event_date,
    (ed_min.first_event_date::date - CURRENT_DATE) AS days_until_event
FROM {{schema}}.event e
LEFT JOIN LATERAL (
    SELECT MIN(ed.start) AS first_event_date
    FROM {{schema}}.event_date ed
    WHERE ed.event_id = e.id
) ed_min ON true
    JOIN {{schema}}.organizer o ON o.id = e.organizer_id
    JOIN {{schema}}.user_organizer_links uol ON uol.organizer_id = o.id
WHERE e.release_status_id < 3
  AND uol.user_id = $1
  AND (
    (e.release_date IS NOT NULL AND e.release_date <= CURRENT_DATE + $2::int)
   OR (ed_min.first_event_date IS NOT NULL AND ed_min.first_event_date::date <= CURRENT_DATE + $3::int)
    )