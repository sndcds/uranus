WITH target_event AS (
    SELECT *
    FROM {{schema}}.event
    WHERE id = $1
)
SELECT
    ed.id AS event_date_id,
    ed.event_id,
    e.release_status,

    TO_CHAR(ed.start_date, 'YYYY-MM-DD') AS start_date,
    TO_CHAR(ed.start_time, 'HH24:MI') AS start_time,
    TO_CHAR(ed.end_date, 'YYYY-MM-DD') AS end_date,
    TO_CHAR(ed.end_time, 'HH24:MI') AS end_time,
    TO_CHAR(ed.entry_time, 'HH24:MI') AS entry_time,
    ed.duration,

    -- Venue logic: prefer event_date.venue_id, fallback to event.venue_id
    v.id AS venue_id,
    v.name AS venue_name,
    v.street AS venue_street,
    v.house_number AS venue_house_number,
    v.postal_code AS venue_postal_code,
    v.city AS venue_city,
    v.country AS venue_country,
    v.state AS venue_state,
    ST_X(v.wkb_pos) AS venue_lon,
    ST_Y(v.wkb_pos) AS venue_lat,
    v.website_url AS venue_url,
    venue_logo.main_logo_image_id AS venue_logo_image_id,

    -- Space logic: take from event_date only if event_date.venue_id exists, else NULL
    s.id AS space_id,
    s.name AS space_name,
    s.total_capacity AS space_total_capacity,
    s.seating_capacity AS space_seating_capacity,
    s.building_level AS space_building_level,
    s.website_url AS space_url,
    s.accessibility_flags::text AS accessibility_flags,
    s.accessibility_summary AS accessibility_summary,

    ed.accessibility_info AS accessibility_info,
    ed.visitor_info_flags

FROM {{schema}}.event_date ed
JOIN target_event e ON ed.event_id = e.id

-- Venue fallback
LEFT JOIN {{schema}}.venue v
ON v.id = COALESCE(ed.venue_id, e.venue_id)

-- Space fallback
LEFT JOIN {{schema}}.space s
ON s.id = CASE
WHEN ed.venue_id IS NOT NULL THEN ed.space_id
ELSE e.space_id
END

LEFT JOIN LATERAL (
    SELECT pi.id AS main_logo_image_id
    FROM uranus.pluto_image_link pil
    JOIN uranus.pluto_image pi
      ON pi.id = pil.pluto_image_id
    WHERE pil.context = 'venue'
      AND pil.context_id = v.id
      AND pil.identifier = 'main_logo'
    LIMIT 1
) venue_logo ON true

ORDER BY ed.start_date, ed.start_time