SELECT
    {{search_rank}},
    edp.event_date_uuid,
    edp.event_uuid,
    ep.org_uuid,
    COALESCE(edp.venue_uuid, ep.venue_uuid) AS venue_uuid,
    COALESCE(edp.space_uuid, ep.space_uuid) AS space_uuid,

    TO_CHAR(
        CASE
            WHEN edp.end_date IS NOT NULL
            AND edp.end_date - edp.start_date > 5
            AND edp.start_date <= $1::date
            AND edp.end_date >= $1::date
            THEN $1::date
            ELSE edp.start_date
        END,
        'YYYY-MM-DD'
    ) AS start_date,

    TO_CHAR(edp.start_time, 'HH24:MI') AS start_time,
    TO_CHAR(edp.end_date, 'YYYY-MM-DD') AS end_date,
    TO_CHAR(edp.end_time, 'HH24:MI') AS end_time,
    TO_CHAR(edp.entry_time, 'HH24:MI') AS entry_time,
    edp.duration,
    edp.all_day,

    CASE
        WHEN edp.release_status IS NULL
        OR edp.release_status = 'inherited'
        THEN ep.release_status
        ELSE edp.release_status
    END AS release_status,

    edp.ticket_link,
    ep.title,
    ep.subtitle,
    ep.summary,
    ep.categories,
    ep.types,
    ep.languages,
    ep.tags,
    ep.org_name,
    ep.image_uuid,
    COALESCE(ep.image_ai_label, 'none') AS image_ai_label,

    COALESCE(edp.venue_name, ep.venue_name) AS venue_name,
    COALESCE(edp.venue_city, ep.venue_city) AS venue_city,
    COALESCE(edp.venue_street, ep.venue_street) AS venue_street,
    COALESCE(edp.venue_house_number, ep.venue_house_number) AS venue_house_number,
    COALESCE(edp.venue_postal_code, ep.venue_postal_code) AS venue_postal_code,
    COALESCE(edp.venue_state, ep.venue_state) AS venue_state,
    COALESCE(edp.venue_country, ep.venue_country) AS venue_country,

    ST_Y(COALESCE(edp.venue_point, ep.venue_point)) AS venue_lat,
    ST_X(COALESCE(edp.venue_point, ep.venue_point)) AS venue_lon,

    COALESCE(edp.venue_name, ep.venue_name) AS venue_name,

    COALESCE(
        edp.space_accessibility_flags,
        ep.space_accessibility_flags
    ) AS space_accessibility_flags,

    ep.min_age,
    ep.max_age,
    ep.price_type,
    ep.min_price,
    ep.max_price,
    ep.currency,
    ep.visitor_info_flags

FROM {{schema}}.event_date_projection edp

JOIN {{schema}}.event_projection ep
    ON ep.event_uuid = edp.event_uuid

{{portal_join}}

WHERE ep.release_status IN (
        'released',
        'cancelled',
        'deferred',
        'rescheduled'
    )

    AND {{date_conditions}}

{{conditions}}
{{portal_conditions}}

ORDER BY
    CASE
        WHEN edp.end_date IS NOT NULL
             AND edp.end_date - edp.start_date > 5
             AND edp.start_date <= $1::date
             AND edp.end_date >= $1::date
            THEN $1::date + TIME '23:59:59'

        ELSE
            edp.start_date
            + COALESCE(edp.start_time, TIME '00:00:00')
END ASC,

    edp.event_date_uuid ASC

{{limit}}