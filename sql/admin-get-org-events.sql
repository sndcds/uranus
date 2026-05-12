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
        ed.duration,
        ed.accessibility_info
    FROM {{schema}}.event_date ed
),

base AS (
    SELECT
        e.uuid AS uuid,
        edt.event_date_uuid AS date_uuid,
        e.title,
        e.subtitle,
        e.org_uuid,
        eo.name AS org_name,

        TO_CHAR(edt.start_date, 'YYYY-MM-DD') AS start_date,
        TO_CHAR(edt.start_time, 'HH24:MI') AS start_time,
        TO_CHAR(edt.end_date, 'YYYY-MM-DD') AS end_date,
        TO_CHAR(edt.end_time, 'HH24:MI') AS end_time,

        e.release_status,
        TO_CHAR(e.release_date, 'YYYY-MM-DD') AS release_date,
        e.categories,

        v.uuid AS venue_uuid,
        v.name AS venue_name,
        s.uuid AS space_uuid,
        s.name AS space_name,

        main_image_link.image_uuid,
        main_image_link.image_url,

        et_data.event_types,

        (   COALESCE((uel.permissions & (1<<25)) <> 0, FALSE)
            OR COALESCE((uol.permissions & (1<<25)) <> 0, FALSE)
            OR COALESCE((uvl.permissions & (1<<25)) <> 0, FALSE)
        ) AS can_edit_event,

        (   COALESCE((uel.permissions & (1<<26)) <> 0, FALSE)
            OR COALESCE((uol.permissions & (1<<26)) <> 0, FALSE)
            OR COALESCE((uvl.permissions & (1<<26)) <> 0, FALSE)
        ) AS can_delete_event,

        (   COALESCE((uel.permissions & (1<<27)) <> 0, FALSE)
            OR COALESCE((uol.permissions & (1<<27)) <> 0, FALSE)
            OR COALESCE((uvl.permissions & (1<<27)) <> 0, FALSE)
        ) AS can_release_event,

        (   COALESCE((uel.permissions & (1<<28)) <> 0, FALSE)
            OR COALESCE((uol.permissions & (1<<28)) <> 0, FALSE)
            OR COALESCE((uvl.permissions & (1<<28)) <> 0, FALSE)
        ) AS can_view_event_insights,

        (online_link IS NOT NULL) AS is_online_event,

        ROW_NUMBER() OVER (
            PARTITION BY e.uuid
            ORDER BY edt.start_date NULLS LAST, edt.start_time NULLS LAST
        ) AS time_series_index,

        COUNT(edt.event_date_uuid) OVER (
            PARTITION BY e.uuid
        ) AS time_series

    FROM {{schema}}.event e
    LEFT JOIN event_data edt ON edt.event_uuid = e.uuid
    LEFT JOIN {{schema}}.venue v ON v.uuid = COALESCE(edt.venue_uuid, e.venue_uuid)
    LEFT JOIN {{schema}}.space s ON s.uuid = (CASE
        WHEN edt.venue_uuid IS NOT NULL THEN edt.space_uuid
        ELSE e.space_uuid
    END)::uuid
    LEFT JOIN {{schema}}.organization eo ON eo.uuid = e.org_uuid

    LEFT JOIN LATERAL (
        SELECT jsonb_agg(event_type ORDER BY event_type.type_id, event_type.genre_id) AS event_types
        FROM (
            SELECT DISTINCT etl.type_id, etl.genre_id
            FROM {{schema}}.event_type_link etl
            WHERE etl.event_uuid = e.uuid
        ) event_type
    ) et_data ON TRUE

    LEFT JOIN LATERAL (
        SELECT
            pil.pluto_image_uuid AS image_uuid,
            format('{{base_api_url}}/api/image/%s', pil.pluto_image_uuid::text) AS image_url
        FROM {{schema}}.pluto_image_link pil
        WHERE pil.context = 'event'
        AND pil.context_uuid = e.uuid
        AND pil.identifier = 'main'
        LIMIT 1
    ) main_image_link ON TRUE

    LEFT JOIN {{schema}}.user_event_link uel
    ON uel.event_uuid = e.uuid
    AND uel.user_uuid = $3::uuid

    INNER JOIN {{schema}}.user_organization_link uol
    ON uol.org_uuid = e.org_uuid
    AND uol.user_uuid = $3::uuid

    LEFT JOIN {{schema}}.user_venue_link uvl
    ON uvl.venue_uuid = e.venue_uuid
    AND uvl.user_uuid = $3::uuid

    WHERE eo.uuid = $1::uuid
    AND (edt.start_date >= $2::date OR edt.start_date IS NULL)
)

SELECT *
FROM base
WHERE can_edit_event OR can_delete_event OR can_release_event OR can_view_event_insights
ORDER BY start_date NULLS LAST, start_time NULLS LAST