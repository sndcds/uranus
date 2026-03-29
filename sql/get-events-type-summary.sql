WITH event_data AS (
    SELECT
        ed.uuid AS event_date_uuid,
        ed.event_uuid,
        ed.venue_uuid,
        ed.space_uuid,
        ed.start_date,
        ed.start_time,
        ed.end_date,
        ed.end_time,
        ed.entry_time,
        ed.duration
    FROM {{schema}}.event_date ed
    JOIN {{schema}}.event e
    ON e.uuid = ed.event_uuid
    AND e.release_status IN ('released', 'cancelled', 'deferred', 'rescheduled')
    {{event-date-conditions}}
),
venue_data AS (
    SELECT
        ed.event_date_uuid AS venue_event_date,
        COALESCE(v_ed.uuid, v_ev.uuid) AS venue_uuid,
        COALESCE(v_ed.name, v_ev.name) AS venue_name,
        COALESCE(v_ed.street, v_ev.street) AS venue_street,
        COALESCE(v_ed.house_number, v_ev.house_number) AS venue_house_number,
        COALESCE(v_ed.postal_code, v_ev.postal_code) AS venue_postal_code,
        COALESCE(v_ed.city, v_ev.city) AS venue_city,
        COALESCE(v_ed.country, v_ev.country) AS venue_country,
        COALESCE(v_ed.state, v_ev.state) AS venue_state,
        COALESCE(ST_X(v_ed.point), ST_X(v_ev.point)) AS venue_lon,
        COALESCE(ST_Y(v_ed.point), ST_Y(v_ev.point)) AS venue_lat,
        COALESCE(v_ed.point, v_ev.point) AS venue_point
    FROM event_data ed
    LEFT JOIN {{schema}}.venue v_ev
    ON v_ev.uuid = (SELECT e.venue_uuid FROM {{schema}}.event e WHERE e.uuid = ed.event_uuid)
    LEFT JOIN {{schema}}.venue v_ed
    ON v_ed.uuid = ed.venue_uuid
)
SELECT json_build_object(
    'total', (SELECT COUNT(*) FROM event_data),
    'organization_summary', (
        SELECT json_agg(organization_summary)
        FROM (
            SELECT
                e.org_uuid AS uuid,
                o.name AS name,
                COUNT(ed.event_date_uuid) AS event_date_count
            FROM event_data ed
            JOIN {{schema}}.event e ON e.uuid = ed.event_uuid
            JOIN {{schema}}.organization o ON o.uuid = e.org_uuid
            GROUP BY e.org_id, o.name
            ORDER BY name ASC
        ) organization_summary
    ),
    'type_summary', (
        SELECT json_agg(type_summary)
        FROM (
            SELECT
                COUNT(etl.event_uuid) AS count,
                et.type_uuid,
                et.name AS type_name
            FROM event_data ed
            JOIN {{schema}}.event_type_link etl ON etl.event_uuid = ed.event_uuid
            JOIN {{schema}}.event_type et ON et.type_id = etl.type_id AND et.iso_639_1 = $1
            GROUP BY et.type_id, et.name
            ORDER BY count DESC
        ) type_summary
    ),
    'venues_summary', (
        SELECT json_agg(venue_summary)
        FROM (
            SELECT
                vd.venue_uuid AS uuid,
                vd.venue_name AS name,
                vd.venue_city AS city,
                COUNT(ed.event_date_uuid) AS event_date_count
            FROM event_data ed
            JOIN venue_data vd ON vd.venue_event_date = ed.event_date_uuid
            WHERE vd.venue_uuid IS NOT NULL
            GROUP BY vd.venue_uuid, vd.venue_name, vd.venue_city
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
                et_data.event_types
            FROM event_data ed
            JOIN venue_data vd ON vd.venue_event_date = ed.event_date_id
            LEFT JOIN LATERAL (
                SELECT pli.id, pli.focus_x, pli.focus_y
                FROM {{schema}}.pluto_image pli
                JOIN {{schema}}.event e ON e.uuid = ed.event_uuid
                WHERE pli.uuid = e.image_ids[1]
                LIMIT 1
            ) image_data ON true

            LEFT JOIN LATERAL (
                SELECT jsonb_agg(
                    DISTINCT jsonb_build_object(
                        'type_id', etl.type_id,
                        'type_name', et.name,
                        'genre_id', COALESCE(gt.type_id, 0),
                        'genre_name', gt.name
                   )
                ) AS event_types
                FROM {{schema}}.event_type_link etl
                JOIN {{schema}}.event_type et
                ON et.type_id = etl.type_id AND et.iso_639_1 = $1
                LEFT JOIN {{schema}}.genre_type gt
                ON gt.genre_id = etl.genre_id AND gt.iso_639_1 = $1
                WHERE etl.event_id = ed.event_id
            ) et_data ON true

            {{conditions}}
            {{order}}
            {{limit}}
        ) AS event_detail
    )
) AS summary