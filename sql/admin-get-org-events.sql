SELECT
    e.uuid AS uuid,
    ed.uuid AS date_uuid,

    CASE
        WHEN ed.release_status IS NULL OR ed.release_status = 'inherited'
            THEN e.release_status
        ELSE ed.release_status
        END
        AS release_status,

    TO_CHAR(e.release_date, 'YYYY-MM-DD') AS release_date,

    e.categories,
    et_data.event_types AS event_types,
    e.title AS event_title,
    e.subtitle AS event_subtitle,
    image.uuid AS image_uuid,
    image.url AS image_url,
    e.online_link,

    o.uuid AS org_uuid,
    o.name AS org_name,
    o.city AS org_city,
    v.uuid AS venue_uuid,
    v.name AS venue_name,
    v.city AS venue_city,
    s.uuid AS space_uuid,
    s.name AS space_name,

    TO_CHAR(ed.start_date, 'YYYY-MM-DD') AS start_date,
    TO_CHAR(ed.start_time, 'HH24:MI') AS start_time,
    TO_CHAR(ed.end_date, 'YYYY-MM-DD') AS end_date,
    TO_CHAR(ed.end_time, 'HH24:MI') AS end_time,

    COALESCE((uol.permissions & (1<<25)) <> 0, FALSE) AS can_edit_event,
    COALESCE((uol.permissions & (1<<26)) <> 0, FALSE) AS can_delete_event,
    COALESCE((uol.permissions & (1<<27)) <> 0, FALSE) AS can_release_event,
    COALESCE((uol.permissions & (1<<28)) <> 0, FALSE) AS can_view_event_insights,

    COUNT(ed.uuid) OVER (PARTITION BY e.uuid) AS series_total,

    ROW_NUMBER() OVER (
	    PARTITION BY e.uuid
        ORDER BY ed.start_date NULLS LAST, ed.start_time NULLS LAST
    ) AS series_index,

    COUNT(ed.uuid) FILTER (
        WHERE ed.start_date >= NOW()
    ) OVER (PARTITION BY e.uuid) AS upcoming_dates_count,

    TO_CHAR(
        MIN(ed.start_date) FILTER (
            WHERE ed.start_date >= NOW()
        ) OVER (PARTITION BY e.uuid),
        'YYYY-MM-DD'
    ) AS next_date

FROM {{schema}}.event e

LEFT JOIN {{schema}}.event_date ed
    ON ed.event_uuid = e.uuid

JOIN {{schema}}.user_organization_link uol
    ON uol.org_uuid = e.org_uuid
        AND uol.user_uuid = $1::uuid
        AND (uol.permissions & (1 << 25)) <> 0

LEFT JOIN {{schema}}.organization o
    ON o.uuid = e.org_uuid

LEFT JOIN {{schema}}.venue v
    ON v.uuid = COALESCE(ed.venue_uuid, e.venue_uuid)

LEFT JOIN {{schema}}.space s
    ON s.uuid = COALESCE(ed.space_uuid, e.space_uuid)

LEFT JOIN LATERAL (
    SELECT COALESCE(
        jsonb_agg(
            event_type
            ORDER BY event_type.type_id, event_type.genre_id
        ),
        '[]'::jsonb
    ) AS event_types
    FROM (
        SELECT DISTINCT
            etl.type_id,
            etl.genre_id
        FROM {{schema}}.event_type_link etl
        WHERE etl.event_uuid = e.uuid
    ) event_type
) et_data ON TRUE


LEFT JOIN LATERAL (
    SELECT
        pil.pluto_image_uuid AS uuid,
        format('{{base_api_url}}/api/image/%s', pil.pluto_image_uuid::text) AS url
    FROM {{schema}}.pluto_image_link pil
    WHERE pil.context = 'event'
        AND pil.context_uuid = e.uuid
        AND pil.identifier = 'main'
    LIMIT 1
) image ON TRUE

WHERE e.org_uuid = $2::uuid
    AND (
        ed.start_date IS NULL
        OR ed.start_date <= CURRENT_DATE
    )
    AND (
        ed.end_date IS NULL
        OR ed.end_date >= CURRENT_DATE
    )

ORDER BY
    (ed.uuid IS NULL) DESC,
    start_date NULLS LAST,
    start_time NULLS LAST