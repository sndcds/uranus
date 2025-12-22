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
        ed.visitor_info_flags
    FROM {{schema}}.event_date ed
    JOIN {{schema}}.event e
ON e.id = ed.event_id
    AND e.release_status_id >= 3
    {{event-date-conditions}}
    ),
    venue_data AS (
SELECT
    ed.event_date_id AS venue_event_date,
    COALESCE(v_ed.id, v_ev.id) AS venue_id,
    COALESCE(v_ed.name, v_ev.name) AS venue_name,
    COALESCE(v_ed.street, v_ev.street) AS venue_street,
    COALESCE(v_ed.house_number, v_ev.house_number) AS venue_house_number,
    COALESCE(v_ed.postal_code, v_ev.postal_code) AS venue_postal_code,
    COALESCE(v_ed.city, v_ev.city) AS venue_city,
    COALESCE(v_ed.country_code, v_ev.country_code) AS venue_country_code,
    COALESCE(v_ed.state_code, v_ev.state_code) AS venue_state_code,
    COALESCE(ST_X(v_ed.wkb_pos), ST_X(v_ev.wkb_pos)) AS venue_lon,
    COALESCE(ST_Y(v_ed.wkb_pos), ST_Y(v_ev.wkb_pos)) AS venue_lat,
    COALESCE(v_ed.wkb_pos, v_ev.wkb_pos) AS venue_wkb_pos
FROM event_data ed
    LEFT JOIN {{schema}}.venue v_ev
ON v_ev.id = (SELECT e.venue_id FROM {{schema}}.event e WHERE e.id = ed.event_id)
    LEFT JOIN {{schema}}.venue v_ed
    ON v_ed.id = ed.venue_id
    )
SELECT json_build_object(
               'total', (SELECT COUNT(*) FROM event_data),
               'organization_summary', (
                   SELECT json_agg(organization_summary)
                   FROM (
                            SELECT
                                e.organization_id AS id,
                                o.name AS name,
                                COUNT(ed.event_date_id) AS event_date_count
                            FROM event_data ed
                                     JOIN {{schema}}.event e ON e.id = ed.event_id
                                JOIN {{schema}}.organization o ON o.id = e.organization_id
                            GROUP BY e.organization_id, o.name
                            ORDER BY name ASC
                        ) organization_summary
               ),
               'type_summary', (
                   SELECT json_agg(type_summary)
                   FROM (
                            SELECT
                                COUNT(etl.event_id) AS count,
                et.type_id,
                et.name AS type_name
                            FROM event_data ed
                                JOIN {{schema}}.event_type_link etl ON etl.event_id = ed.event_id
                                JOIN {{schema}}.event_type et ON et.type_id = etl.type_id AND et.iso_639_1 = $1
                            GROUP BY et.type_id, et.name
                            ORDER BY count DESC
                        ) type_summary
               ),
               'venues_summary', (
                   SELECT json_agg(venue_summary)
                   FROM (
                            SELECT
                                vd.venue_id AS id,
                                vd.venue_name AS name,
                                vd.venue_city AS city,
                                COUNT(ed.event_date_id) AS event_date_count
                            FROM event_data ed
                                     JOIN venue_data vd ON vd.venue_event_date = ed.event_date_id
                            WHERE vd.venue_id IS NOT NULL
                            GROUP BY vd.venue_id, vd.venue_name, vd.venue_city
                            ORDER BY vd.venue_name ASC
                        ) venue_summary
               ),
               'event_details', (
                   SELECT json_agg(event_detail)
                   FROM (
                            SELECT
                                ed.*,
                                vd.venue_name,
                                vd.venue_city,
                                -- main image
                                image_data.id AS image_id,
                                image_data.focus_x,
                                image_data.focus_y,
                                -- event types
                                et_data.event_types,
                                -- visitor info flags
                                vis_flags.visitor_info_flag_names
                            FROM event_data ed
                            JOIN venue_data vd ON vd.venue_event_date = ed.event_date_id

                            LEFT JOIN LATERAL (
                                SELECT pli.id, pli.focus_x, pli.focus_y
                                FROM {{schema}}.pluto_image pli
                                JOIN {{schema}}.event e ON e.id = ed.event_id
                            WHERE pli.id = e.image1_id
                                LIMIT 1
                        ) image_data ON true

                   LEFT JOIN LATERAL (
                   SELECT jsonb_agg(DISTINCT jsonb_build_object(
                   'type_id', etl.type_id,
                   'type_name', et.name,
                   'genre_id', COALESCE(gt.type_id, 0),
                   'genre_name', gt.name
                   )) AS event_types
                   FROM {{schema}}.event_type_link etl
                   JOIN {{schema}}.event_type et
                   ON et.type_id = etl.type_id AND et.iso_639_1 = $1
                   LEFT JOIN {{schema}}.genre_type gt
                   ON gt.type_id = etl.genre_id AND gt.iso_639_1 = $1
                   WHERE etl.event_id = ed.event_id
                   ) et_data ON true

                   LEFT JOIN LATERAL (
                   SELECT jsonb_agg(name) AS visitor_info_flag_names
                   FROM {{schema}}.visitor_information_flags f
                   WHERE (ed.visitor_info_flags & (1::BIGINT << f.flag)) = (1::BIGINT << f.flag)
                   AND f.iso_639_1 = $1
               ) vis_flags ON true

            {{conditions}}
            {{order}}
            {{limit}}
        ) AS event_detail
    )
) AS summary