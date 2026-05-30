WITH RECURSIVE week_days AS (
    SELECT {{week_start}}::date AS day
UNION ALL
SELECT day + 1
FROM week_days
WHERE day < {{week_end}}::date
    ),

    base AS (
SELECT
    edp.event_date_uuid,
    edp.event_uuid,
    ep.org_uuid,

    (edp.start_date)::date AS event_day,
    edp.start_date,
    edp.start_time,

    COALESCE(edp.venue_uuid, ep.venue_uuid) AS venue_uuid,
    COALESCE(edp.space_uuid, ep.space_uuid) AS space_uuid,

    ep.title,
    ep.subtitle,
    ep.types,
    ep.image_uuid,

    COALESCE(edp.venue_name, ep.venue_name) AS venue_name,
    COALESCE(edp.venue_city, ep.venue_city) AS venue_city,

    ROW_NUMBER() OVER (
    PARTITION BY (edp.start_date)::date
    ORDER BY edp.start_date, edp.start_time, edp.event_date_uuid
    ) AS rn

FROM {{schema}}.event_date_projection edp
    JOIN {{schema}}.event_projection ep
ON ep.event_uuid = edp.event_uuid

    {{portal_join}}

WHERE ep.release_status IN ('released', 'cancelled', 'deferred', 'rescheduled')
  AND edp.start_date >= {{week_start}}::date
  AND edp.start_date <= {{week_end}}::date
    {{conditions}}
    {{portal_conditions}}
    ),

    daily_counts AS (
SELECT
    event_day,
    COUNT(*) AS total_count
FROM base
GROUP BY event_day
    ),

    top_events AS (
SELECT *
FROM base
WHERE rn <= 10
    )

SELECT
    d.day AS event_day,

    COALESCE(
            jsonb_agg(
                    jsonb_build_object(
                            'event_date_uuid', t.event_date_uuid,
                            'event_uuid', t.event_uuid,
                            'org_uuid', t.org_uuid,
                            'start_date', t.start_date,
                            'start_time', t.start_time,
                            'title', t.title,
                            'subtitle', t.subtitle,
                            'types', t.types,
                            'image_uuid', t.image_uuid,
                            'venue_name', t.venue_name,
                            'venue_city', t.venue_city
                    )
                        ORDER BY t.start_time, t.event_date_uuid
            ) FILTER (WHERE t.event_date_uuid IS NOT NULL),
            '[]'::jsonb
    ) AS events,

    COALESCE(dc.total_count, 0) - LEAST(COALESCE(dc.total_count, 0), 10) AS more_count

FROM week_days d

         LEFT JOIN top_events t
                   ON t.event_day = d.day

         LEFT JOIN daily_counts dc
                   ON dc.event_day = d.day

GROUP BY d.day, dc.total_count

ORDER BY d.day