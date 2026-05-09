SELECT
    edp.event_date_uuid,
    edp.event_uuid,
    ep.org_uuid,
    COALESCE(edp.venue_uuid, ep.venue_uuid) AS venue_uuid,
    COALESCE(edp.space_uuid, ep.space_uuid) AS space_uuid,
    TO_CHAR(edp.start_date, 'YYYY-MM-DD') AS start_date,
    TO_CHAR(edp.start_time, 'HH24:MI') AS start_time,
    TO_CHAR(edp.end_date, 'YYYY-MM-DD') AS end_date,
    TO_CHAR(edp.end_time, 'HH24:MI') AS end_time,
    TO_CHAR(edp.entry_time, 'HH24:MI') AS entry_time,
    edp.duration,
    edp.all_day,
    ep.release_status,
    edp.ticket_link,
    ep.title,
    ep.subtitle,
    ep.categories,
    ep.types,
    ep.languages,
    ep.tags,
    ep.org_name,
    ep.image_uuid,
    COALESCE(edp.venue_name, ep.venue_name) AS venue_name,
    COALESCE(edp.venue_city, ep.venue_city) AS venue_city,
    COALESCE(edp.venue_street, ep.venue_street) AS venue_street,
    COALESCE(edp.venue_house_number, ep.venue_house_number) AS venue_house_number,
    COALESCE(edp.venue_postal_code, ep.venue_postal_code) AS venue_postal_code,
    COALESCE(edp.venue_state, ep.venue_state) AS venue_state,
    COALESCE(edp.venue_country, ep.venue_country) AS venue_country,
    ST_Y(COALESCE(edp.venue_point, ep.venue_point)) AS venue_lat,
    ST_X(COALESCE(edp.venue_point, ep.venue_point)) AS venue_lon,
    COALESCE(edp.space_name, ep.space_name) AS space_name,
    COALESCE(edp.space_accessibility_flags, ep.space_accessibility_flags) AS space_accessibility_flags,
    ep.min_age,
    ep.max_age,
    ep.visitor_info_flags
FROM {{schema}}.event_date_projection edp
JOIN {{schema}}.event_projection ep
{{portal_join}}
ON ep.event_uuid = edp.event_uuid
WHERE ep.release_status IN ('released', 'cancelled', 'deferred', 'rescheduled')
AND {{date_conditions}}
{{conditions}}
{{portal_conditions}}
ORDER BY edp.event_start_at ASC, edp.event_date_uuid ASC
{{limit}}