WITH event_check AS (
    SELECT
        e.uuid,
        e.title,
        e.org_uuid,

        CASE
            WHEN e.release_status IN ('draft', 'review')
                THEN e.release_status
            ELSE NULL
            END AS release_status,

        e.online_link,

        COALESCE(
                e.venue_uuid,
                (
                    ARRAY_AGG(ed.venue_uuid)
                        FILTER (WHERE ed.venue_uuid IS NOT NULL)
                    )[1]
        ) AS venue_uuid,

        -- first upcoming date
        MIN(ed.start_date) FILTER (
            WHERE ed.start_date >= CURRENT_DATE
        ) AS first_date,

        COUNT(ed.uuid) AS event_date_count,

        -- image exists?
        EXISTS (
            SELECT 1
            FROM {{schema}}.pluto_image_link pil
            WHERE pil.context = 'event'
              AND pil.context_uuid = e.uuid
              AND pil.identifier = 'main'
        ) AS has_image,

        -- venue exists on event OR any date
        (
            e.venue_uuid IS NOT NULL
                OR EXISTS (
                SELECT 1
                FROM {{schema}}.event_date ed2
                WHERE ed2.event_uuid = e.uuid
                  AND ed2.venue_uuid IS NOT NULL
            )
            ) AS has_venue

    FROM {{schema}}.event e

             LEFT JOIN {{schema}}.event_date ed
                       ON ed.event_uuid = e.uuid

    WHERE e.org_uuid = $2::uuid

GROUP BY
    e.uuid,
    e.title,
    e.org_uuid,
    e.release_status,
    e.online_link,
    e.venue_uuid
    )

SELECT
    ec.uuid AS uuid,
    ec.title AS title,
    o.uuid AS org_uuid,
    o.name AS org_name,
    v.uuid AS venue_uuid,
    v.name AS venue_name,
    v.city AS venue_city,
    ec.release_status,
    ec.first_date AS first_date,
    (ec.first_date - CURRENT_DATE) AS days_until_first_date,

    -- QA flags
    NOT ec.has_image AS no_image,

    (ec.event_date_count = 0) AS no_event_dates,

    (
        NOT ec.has_venue
            AND COALESCE(NULLIF(TRIM(ec.online_link), ''), '') = ''
        ) AS no_venue_or_online_link,

    NOT EXISTS (
        SELECT 1
        FROM {{schema}}.event_type_link etl
        WHERE etl.event_uuid = ec.uuid
    ) AS no_event_type,

    COALESCE(TRIM(ec.title), '') = '' AS no_title,

    ec.first_date IS NULL AS no_upcoming_date

FROM event_check ec

         JOIN {{schema}}.user_organization_link uol
              ON uol.user_uuid = $1::uuid
		AND uol.org_uuid = $2::uuid

LEFT JOIN {{schema}}.organization o
ON o.uuid = ec.org_uuid

    LEFT JOIN {{schema}}.venue v
    ON v.uuid = ec.venue_uuid

WHERE
    ec.release_status IN ('draft', 'review')
  AND (
    ec.event_date_count = 0
    OR ec.first_date <= CURRENT_DATE + ($3 * interval '1 day')
    )

ORDER BY
    ec.first_date NULLS FIRST,
    ec.title