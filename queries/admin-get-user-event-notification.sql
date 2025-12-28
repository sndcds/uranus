SELECT
    e.id AS event_id,
    e.title AS event_title,
    e.organization_id,
    o.name AS organization_name,
    e.release_date,
    e.release_status_id,
    es.name AS release_status_name,
    ed_min.first_event_date::date AS earliest_event_date,
    ed_max.last_event_date::date AS latest_event_date,
    (e.release_date - CURRENT_DATE) AS days_until_release,
    GREATEST(ed_min.first_event_date::date - CURRENT_DATE, 0) AS days_until_event
FROM {{schema}}.event e
LEFT JOIN LATERAL (
    SELECT MIN(ed.start_date) AS first_event_date
    FROM {{schema}}.event_date ed
    WHERE ed.event_id = e.id
) ed_min ON true
LEFT JOIN LATERAL (
    SELECT MAX(ed.start_date) AS last_event_date
    FROM {{schema}}.event_date ed
    WHERE ed.event_id = e.id
) ed_max ON true
    JOIN {{schema}}.organization o ON o.id = e.organization_id
    JOIN {{schema}}.user_organization_link uol ON uol.organization_id = o.id
    JOIN {{schema}}.event_status es
    ON es.status_id = e.release_status_id
    AND es.iso_639_1 = $4
WHERE ed_max.last_event_date >= NOW()
AND e.release_status_id < 3
AND uol.user_id = $1
AND (
    (e.release_date IS NOT NULL AND e.release_date <= CURRENT_DATE + $2::int)
    OR (ed_min.first_event_date IS NOT NULL AND ed_min.first_event_date::date <= CURRENT_DATE + $3::int)
)