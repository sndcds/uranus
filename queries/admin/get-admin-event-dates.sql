-- SqlGetEventDates
SELECT
    ed.id AS event_date_id,
    ed.event_id,
    TO_CHAR(ed.start, 'YYYY-MM-DD') AS start_date,
    TO_CHAR(ed.start, 'HH24:MI') AS start_time,
    TO_CHAR(ed."end", 'YYYY-MM-DD') AS end_date,
    TO_CHAR(ed."end", 'HH24:MI') AS end_time,
    TO_CHAR(ed.entry_time, 'HH24:MI') AS entry_time,
    ed.duration,
    ed.accessibility_flags,
    ed.visitor_info_flags,

    -- Venue fallback
    v.id AS venue_id,
    v.name AS venue_name,
    v.street AS venue_street,
    v.house_number AS venue_house_number,
    v.postal_code AS venue_postal_code,
    v.city AS venue_city,
    v.state_code AS venue_state_code,
    v.country_code AS venue_country_code,
    ST_X(v.wkb_geometry) AS venue_lon,
    ST_Y(v.wkb_geometry) AS venue_lat,

    -- Space fallback
    space_data.id AS space_id,
    space_data.name AS space_name,
    space_data.total_capacity AS space_total_capacity,
    space_data.seating_capacity AS space_seating_capacity,
    space_data.building_level AS space_building_level,
    space_data.website_url AS space_url

FROM {{schema}}.event_date ed
JOIN {{schema}}.event e ON ed.event_id = e.id

-- Venue fallback: use event_date.venue_id if exists, otherwise event.venue_id
    LEFT JOIN {{schema}}.venue v ON v.id = COALESCE(ed.venue_id, e.venue_id)

-- Space fallback: lateral join to pick event_date.space_id or fallback to event.space_id
    LEFT JOIN LATERAL (
    SELECT *
    FROM {{schema}}.space s2
    WHERE s2.id = COALESCE(ed.space_id, e.space_id)
    LIMIT 1
    ) space_data ON TRUE

WHERE e.id = $1
ORDER BY ed.start;