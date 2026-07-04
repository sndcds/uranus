WITH week_days AS (
    SELECT generate_series(
       '{{week_start}}'::date,
       '{{week_end}}'::date,
       interval '1 day'
   )::date AS day
),

base AS (
    SELECT
        edp.event_date_uuid,
        edp.event_uuid,
        ep.org_uuid,

        edp.start_date::date AS event_day,
        edp.start_date,
        edp.start_time,

        COALESCE(edp.venue_uuid, ep.venue_uuid) AS venue_uuid,
        COALESCE(edp.space_uuid, ep.space_uuid) AS space_uuid,

        ep.title,
        ep.subtitle,
        ep.types,
        ep.image_uuid,

        ep.categories,

        CASE
            WHEN edp.release_status IS NULL OR edp.release_status = 'inherited'
            THEN ep.release_status
            ELSE edp.release_status
        END AS release_status,

        COALESCE(edp.venue_name, ep.venue_name) AS venue_name,
        COALESCE(edp.venue_city, ep.venue_city) AS venue_city,

        ROW_NUMBER() OVER (
           PARTITION BY edp.start_date::date
            ORDER BY edp.start_date, edp.start_time, edp.event_date_uuid
        ) AS rn

    FROM {{schema}}.event_date_projection edp
    JOIN {{schema}}.event_projection ep
        ON ep.event_uuid = edp.event_uuid

    {{portal_join}}

    WHERE ep.release_status IN ('released', 'cancelled', 'deferred', 'rescheduled')
        AND edp.start_date >= '{{week_start}}'::date
        AND edp.start_date <= '{{week_end}}'::date
        {{conditions}}
        {{portal_conditions}}
),

daily_counts AS (
    SELECT event_day, COUNT(*) AS total_count
    FROM base
    GROUP BY event_day
),

events_agg AS (
    SELECT
        event_day,
        jsonb_agg(
            jsonb_build_object(
                'uuid', event_uuid,
                'date_uuid', event_date_uuid,
                'date_slug', to_char(start_date, 'YYYYMMDD') || to_char(start_time, 'HH24MI'),
                'org_uuid', org_uuid,
                'start_date', start_date,
                'start_time', start_time,
                'title', title,
                'subtitle', subtitle,
                'types', types,
                'image_uuid', image_uuid,
                'categories', categories,
                'release_status', release_status,
                'venue_name', venue_name,
                'venue_city', venue_city
            )
            ORDER BY start_time, event_date_uuid
        ) AS events
    FROM base
    WHERE rn <= 15
    GROUP BY event_day
)

SELECT
    TO_CHAR(d.day, 'YYYY-MM-DD') AS event_day,
    COALESCE(e.events, '[]'::jsonb) AS events,
    GREATEST(COALESCE(dc.total_count, 0) - 15, 0) AS more_count

FROM week_days d
LEFT JOIN events_agg e ON e.event_day = d.day
LEFT JOIN daily_counts dc ON dc.event_day = d.day
ORDER BY d.day