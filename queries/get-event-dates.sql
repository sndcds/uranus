WITH target_event AS (
    SELECT *
    FROM {{schema}}.event
WHERE id = $1
    )
SELECT
    ed.id AS event_date_id,
    ed.event_id,
    ed.start,
    ed."end" AS end_time,
    ed.entry_time,
    ed.duration,
    ed.accessibility_flags,
    ed.visitor_info_flags,

    -- Venue logic: prefer event_date.venue_id, fallback to event.venue_id
    v.id AS venue_id,
    v.name AS venue_name,
    v.street AS venue_street,
    v.house_number AS venue_house_number,
    v.postal_code AS venue_postal_code,
    v.city AS venue_city,
    v.country_code AS venue_country,
    v.state_code AS venue_state,
    ST_AsText(v.wkb_geometry) AS venue_geometry,
    ST_X(v.wkb_geometry) AS venue_lon,
    ST_Y(v.wkb_geometry) AS venue_lat,

    -- Space logic: prefer event_date.space_id, fallback to event.space_id
    s.id AS space_id,
    s.name AS space_name,
    s.total_capacity AS space_total_capacity,
    s.seating_capacity AS space_seating_capacity,
    s.building_level AS space_building_level,
    s.website_url AS space_url

FROM {{schema}}.event_date ed
JOIN target_event e ON ed.event_id = e.id

-- Venue fallback
    LEFT JOIN {{schema}}.venue v
    ON v.id = COALESCE(ed.venue_id, e.venue_id)

-- Space fallback
    LEFT JOIN {{schema}}.space s
    ON s.id = COALESCE(ed.space_id, e.space_id)

ORDER BY ed.start;