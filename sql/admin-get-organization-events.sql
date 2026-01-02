WITH event_data AS (
    SELECT
        ed.id AS event_date_id,
        ed.event_id,
        ed.venue_id,
        ed.space_id,
        ed.start_date,
        ed.start_time,
        ed.end_date,
        ed.end_time,
        ed.entry_time,
        ed.duration,
        ed.accessibility_info,
        ed.visitor_info_flags
    FROM {{schema}}.event_date ed
)
SELECT
    e.id AS event_id,
    event_date_id,
    e.title AS event_title,
    e.subtitle AS event_subtitle,
    e.organization_id AS event_organization_id,
    eo.name AS event_organization_name,
    TO_CHAR(edt.start_date, 'YYYY-MM-DD') AS start_date,
    TO_CHAR(edt.start_time, 'HH24:MI') AS start_time,
    TO_CHAR(edt.end_date, 'YYYY-MM-DD') AS end_date,
    TO_CHAR(edt.end_time, 'HH24:MI') AS end_time,
    e.release_status_id,
    TO_CHAR(e.release_date, 'YYYY-MM-DD') AS release_date,
    v.id AS venue_id,
    v.name AS venue_name,
    s.id AS space_id,
    s.name AS space_name,
    el.id AS location_id,
    el.name AS location_name,
    e.image1_id AS image_id,
    et_data.event_types,

    -- Permissions via bitmask
    COALESCE((uel.permissions & (1<<25)) <> 0, FALSE)
        OR COALESCE((uol.permissions & (1<<25)) <> 0, FALSE)
        OR COALESCE((uvl.permissions & (1<<25)) <> 0, FALSE) AS can_edit_event,
    COALESCE((uel.permissions & (1<<26)) <> 0, FALSE)
        OR COALESCE((uol.permissions & (1<<26)) <> 0, FALSE)
        OR COALESCE((uvl.permissions & (1<<26)) <> 0, FALSE) AS can_delete_event,
    COALESCE((uel.permissions & (1<<27)) <> 0, FALSE)
        OR COALESCE((uol.permissions & (1<<27)) <> 0, FALSE)
        OR COALESCE((uvl.permissions & (1<<27)) <> 0, FALSE) AS can_release_event,

    ROW_NUMBER() OVER (
            PARTITION BY edt.event_id
            ORDER BY edt.start_date, edt.start_time
        ) AS time_series_index,
    COUNT(edt.event_date_id) OVER (PARTITION BY edt.event_id) AS time_series

FROM event_data edt
LEFT JOIN {{schema}}.event e ON edt.event_id = e.id
LEFT JOIN {{schema}}.venue v ON COALESCE(edt.venue_id, e.venue_id) = v.id
LEFT JOIN {{schema}}.space s ON COALESCE(edt.space_id, e.space_id) = s.id
LEFT JOIN {{schema}}.organization o ON v.organization_id = o.id
LEFT JOIN {{schema}}.organization eo ON e.organization_id = eo.id
LEFT JOIN {{schema}}.event_location el ON e.location_id = el.id

LEFT JOIN LATERAL (
    SELECT jsonb_agg(
        event_type ORDER BY event_type.type_id, event_type.genre_id
    ) AS event_types
    FROM (
        SELECT DISTINCT
            etl.type_id,
            etl.genre_id
        FROM {{schema}}.event_type_link etl
        WHERE etl.event_id = e.id
    ) event_type
) et_data ON true

-- User links with permissions bitmask
LEFT JOIN {{schema}}.user_event_link uel ON uel.event_id = e.id AND uel.user_id = $3
LEFT JOIN {{schema}}.user_organization_link uol ON uol.organization_id = e.organization_id AND uol.user_id = $3
LEFT JOIN {{schema}}.user_venue_link uvl ON uvl.venue_id = e.venue_id AND uvl.user_id = $3

WHERE eo.id = $1
AND edt.start_date >= $2::date
ORDER BY edt.start_date, edt.start_time