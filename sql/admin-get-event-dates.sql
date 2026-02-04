SELECT
    ed.id AS event_date_id,
    ed.event_id,
    TO_CHAR(ed.start_date, 'YYYY-MM-DD') AS start_date,
    TO_CHAR(ed.start_time, 'HH24:MI') AS start_time,
    TO_CHAR(ed.end_date, 'YYYY-MM-DD') AS end_date,
    TO_CHAR(ed.end_time, 'HH24:MI') AS end_time,
    TO_CHAR(ed.entry_time, 'HH24:MI') AS entry_time,
    ed.duration,
    ed.all_day,
    ed.accessibility_info,
    ed.venue_id AS venue_id,

    v.name AS venue_name,
    v.street AS venue_street,
    v.house_number AS venue_house_number,
    v.postal_code AS venue_postal_code,
    v.city AS venue_city,
    v.country AS venue_country,
    v.state AS venue_state,
    ST_X(v.wkb_pos) AS venue_lon,
    ST_Y(v.wkb_pos) AS venue_lat,
    v.website_link AS venue_link,

    space_data.id AS space_id,
    space_data.name AS space_name,
    space_data.total_capacity AS space_total_capacity,
    space_data.seating_capacity AS space_seating_capacity,
    space_data.building_level AS space_building_level,
    space_data.website_link AS space_link

FROM {{schema}}.event_date ed
JOIN {{schema}}.event e ON ed.event_id = e.id
LEFT JOIN {{schema}}.venue v ON v.id = COALESCE(ed.venue_id, e.venue_id)
LEFT JOIN LATERAL (
    SELECT *
    FROM {{schema}}.space s2
    WHERE s2.id = CASE
        WHEN ed.venue_id IS NOT NULL THEN ed.space_id
        ELSE NULL
    END
    LIMIT 1
) space_data ON TRUE

WHERE e.id = $1
ORDER BY ed.start_date, ed.start_time