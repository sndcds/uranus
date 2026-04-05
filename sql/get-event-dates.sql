WITH target_event AS (
    SELECT *
    FROM {{schema}}.event
    WHERE uuid = $1::uuid
)
SELECT
    ed.uuid AS event_date_uuid,
    ed.event_uuid,
    e.release_status,

    TO_CHAR(ed.start_date, 'YYYY-MM-DD') AS start_date,
    TO_CHAR(ed.start_time, 'HH24:MI') AS start_time,
    TO_CHAR(ed.end_date, 'YYYY-MM-DD') AS end_date,
    TO_CHAR(ed.end_time, 'HH24:MI') AS end_time,
    TO_CHAR(ed.entry_time, 'HH24:MI') AS entry_time,
    ed.duration,

    -- Venue logic: prefer event_date.venue_uuid, fallback to event.venue_uuid
    v.uuid AS venue_uuid,
    v.name AS venue_name,
    v.street AS venue_street,
    v.house_number AS venue_house_number,
    v.postal_code AS venue_postal_code,
    v.city AS venue_city,
    v.country AS venue_country,
    v.state AS venue_state,
    ST_X(v.point) AS venue_lon,
    ST_Y(v.point) AS venue_lat,
    v.web_link AS venue_link,
    venue_logo.main_logo_uuid AS venue_logo_uuid,
    light_theme_logo.light_theme_logo_uuid AS venue_light_theme_logo_uuid,
    dark_theme_logo.dark_theme_logo_uuid AS venue_dark_theme_logo_uuid,

    -- Space logic: take from event_date only if event_date.venue_uuid exists, else NULL
    s.uuid AS space_uuid,
    s.name AS space_name,
    s.total_capacity AS space_total_capacity,
    s.seating_capacity AS space_seating_capacity,
    s.building_level AS space_building_level,
    s.web_link AS space_link,
    s.accessibility_flags::text AS accessibility_flags,
    s.accessibility_summary AS accessibility_summary,
    ed.accessibility_info AS accessibility_info

FROM {{schema}}.event_date ed
JOIN target_event e ON ed.event_uuid = e.uuid

-- Venue fallback
LEFT JOIN {{schema}}.venue v
ON v.uuid = COALESCE(ed.venue_uuid, e.venue_uuid)

-- Space fallback
LEFT JOIN {{schema}}.space s
ON s.uuid = CASE
WHEN ed.venue_uuid IS NOT NULL THEN ed.space_uuid
ELSE e.space_uuid
END

-- Main logo
LEFT JOIN LATERAL (
    SELECT pi.uuid AS main_logo_uuid
    FROM {{schema}}.pluto_image_link pil
    JOIN {{schema}}.pluto_image pi
      ON pi.uuid = pil.pluto_image_uuid
    WHERE pil.context = 'venue'
      AND pil.context_uuid = v.uuid
      AND pil.identifier = 'main_logo'
    LIMIT 1
) venue_logo ON true

-- Light theme logo
LEFT JOIN LATERAL (
    SELECT pi.uuid AS light_theme_logo_uuid
    FROM {{schema}}.pluto_image_link pil
    JOIN {{schema}}.pluto_image pi
      ON pi.uuid = pil.pluto_image_uuid
    WHERE pil.context = 'venue'
      AND pil.context_uuid = v.uuid
      AND pil.identifier = 'light_theme_logo'
    LIMIT 1
) light_theme_logo ON true

-- Dark theme logo
LEFT JOIN LATERAL (
    SELECT pi.uuid AS dark_theme_logo_uuid
    FROM {{schema}}.pluto_image_link pil
    JOIN {{schema}}.pluto_image pi
      ON pi.uuid = pil.pluto_image_uuid
    WHERE pil.context = 'venue'
      AND pil.context_uuid = v.uuid
      AND pil.identifier = 'dark_theme_logo'
    LIMIT 1
) dark_theme_logo ON true

ORDER BY ed.start_date, ed.start_time