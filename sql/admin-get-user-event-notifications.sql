WITH user_permissions AS (
    SELECT
        (COALESCE(uol.permissions, 0) | COALESCE(uvl.permissions, 0)) AS permissions
    FROM {{schema}}.user u
    LEFT JOIN {{schema}}.user_organization_link uol ON uol.user_uuid = u.uuid
    LEFT JOIN {{schema}}.user_venue_link uvl ON uvl.user_uuid = u.uuid
    WHERE u.uuid = $1::uuid
    LIMIT 1
)

SELECT
    e.uuid AS event_uuid,
    e.title AS event_title,
    e.org_uuid,
    o.name AS org_name,
    e.release_date,
    e.release_status,
    ed_min.first_event_date::date AS earliest_event_date,
    ed_max.last_event_date::date AS latest_event_date,
    (e.release_date - CURRENT_DATE) AS days_until_release,
    GREATEST(ed_min.first_event_date::date - CURRENT_DATE, 0) AS days_until_event
FROM {{schema}}.event e

LEFT JOIN LATERAL (
    SELECT MIN(ed.start_date) AS first_event_date
    FROM {{schema}}.event_date ed
    WHERE ed.event_uuid = e.uuid
) ed_min ON true

    LEFT JOIN LATERAL (
    SELECT MAX(ed.start_date) AS last_event_date
    FROM {{schema}}.event_date ed
    WHERE ed.event_uuid = e.uuid
) ed_max ON true

JOIN {{schema}}.organization o ON o.uuid = e.org_uuid
JOIN {{schema}}.user_organization_link uol ON uol.org_uuid = o.uuid

CROSS JOIN user_permissions up

WHERE ed_max.last_event_date >= NOW()
AND e.release_status IN ('draft', 'review')
AND uol.user_uuid = $1
AND (
    (e.release_date IS NOT NULL AND e.release_date <= CURRENT_DATE + $2::int)
    OR (ed_min.first_event_date IS NOT NULL AND ed_min.first_event_date::date <= CURRENT_DATE + $3::int)
)

AND (
    CASE
        WHEN $4 = 'all' THEN
            (up.permissions & $5::bigint) = $5::bigint
        ELSE
            (up.permissions & $5::bigint) <> 0
        END
)