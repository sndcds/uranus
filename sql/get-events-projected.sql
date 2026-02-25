WITH upcoming_dates AS (
    SELECT
        event_date_id,
        event_id,
        start_date,
        start_time,
        end_date,
        end_time,
        entry_time,
        duration,
        all_day,
        ticket_link,
        venue_id,
        space_id,
        venue_name,
        venue_city,
        venue_street,
        venue_house_number,
        venue_postal_code,
        venue_state,
        venue_country,
        venue_geo_pos,
        space_name,
        space_type,
        space_accessibility_flags,
        event_start_at,
        event_end_at
    FROM {{schema}}.event_date_projection
    WHERE start_date >= CURRENT_DATE
)
SELECT
    edp.event_date_id,
    edp.event_id,
    ep.organization_id,
    COALESCE(edp.venue_id, ep.venue_id) AS venue_id,
    COALESCE(edp.space_id, ep.space_id) AS space_id,
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
    ep.types,
    ep.languages,
    ep.tags,
    ep.organization_name,
    ep.image_id,
    COALESCE(edp.venue_name, ep.venue_name) AS venue_name,
    COALESCE(edp.venue_city, ep.venue_city) AS venue_city,
    COALESCE(edp.venue_street, ep.venue_street) AS venue_street,
    COALESCE(edp.venue_house_number, ep.venue_house_number) AS venue_house_number,
    COALESCE(edp.venue_postal_code, ep.venue_postal_code) AS venue_postal_code,
    COALESCE(edp.venue_state, ep.venue_state) AS venue_state,
    COALESCE(edp.venue_country, ep.venue_country) AS venue_country,
    ST_Y(COALESCE(edp.venue_geo_pos, ep.venue_geo_pos)) AS venue_lat,
    ST_X(COALESCE(edp.venue_geo_pos, ep.venue_geo_pos)) AS venue_lon,
    COALESCE(edp.space_name, ep.space_name) AS space_name,
    COALESCE(edp.space_accessibility_flags, ep.space_accessibility_flags) AS space_accessibility_flags,
    ep.min_age,
    ep.max_age,
    ep.visitor_info_flags
FROM upcoming_dates edp
JOIN {{schema}}.event_projection ep ON ep.event_id = edp.event_id
WHERE ep.release_status IN ('released', 'cancelled', 'deferred', 'rescheduled')
AND {{date_conditions}}
{{conditions}}
ORDER BY edp.event_start_at ASC, edp.event_date_id ASC
{{limit}}