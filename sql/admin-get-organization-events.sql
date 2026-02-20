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
        ed.accessibility_info
    FROM {{schema}}.event_date ed
)

SELECT
    e.id AS id,
    edt.event_date_id AS date_id,
    e.title,
    e.subtitle,
    e.organization_id,
    eo.name AS organization_name,

    TO_CHAR(edt.start_date, 'YYYY-MM-DD') AS start_date,
    TO_CHAR(edt.start_time, 'HH24:MI') AS start_time,
    TO_CHAR(edt.end_date, 'YYYY-MM-DD') AS end_date,
    TO_CHAR(edt.end_time, 'HH24:MI') AS end_time,

    e.release_status,
    TO_CHAR(e.release_date, 'YYYY-MM-DD') AS release_date,

    v.id AS venue_id,
    v.name AS venue_name,
    s.id AS space_id,
    s.name AS space_name,

    main_image_link.image_id,
    main_image_link.image_url,

    et_data.event_types,

    -- Permissions via bitmask
    (
        COALESCE((uel.permissions & (1<<25)) <> 0, FALSE)
            OR COALESCE((uol.permissions & (1<<25)) <> 0, FALSE)
            OR COALESCE((uvl.permissions & (1<<25)) <> 0, FALSE)
    ) AS can_edit_event,

    (
        COALESCE((uel.permissions & (1<<26)) <> 0, FALSE)
            OR COALESCE((uol.permissions & (1<<26)) <> 0, FALSE)
            OR COALESCE((uvl.permissions & (1<<26)) <> 0, FALSE)
    ) AS can_delete_event,

    (
        COALESCE((uel.permissions & (1<<27)) <> 0, FALSE)
            OR COALESCE((uol.permissions & (1<<27)) <> 0, FALSE)
            OR COALESCE((uvl.permissions & (1<<27)) <> 0, FALSE)
    ) AS can_release_event,

    -- Time series
    ROW_NUMBER() OVER (
        PARTITION BY e.id
        ORDER BY edt.start_date NULLS LAST, edt.start_time NULLS LAST
    ) AS time_series_index,

    COUNT(edt.event_date_id) OVER (
        PARTITION BY e.id
    ) AS time_series

FROM {{schema}}.event e

-- Attach dates (optional)
LEFT JOIN event_data edt
ON edt.event_id = e.id

-- Venue resolution (date overrides event default)
LEFT JOIN {{schema}}.venue v
ON v.id = COALESCE(edt.venue_id, e.venue_id)

-- Space resolution
LEFT JOIN {{schema}}.space s
ON s.id = CASE
WHEN edt.venue_id IS NOT NULL THEN edt.space_id
ELSE e.space_id
END

LEFT JOIN {{schema}}.organization eo
ON eo.id = e.organization_id

-- Event types
LEFT JOIN LATERAL (
    SELECT jsonb_agg(event_type ORDER BY event_type.type_id, event_type.genre_id) AS event_types
    FROM (
        SELECT DISTINCT
            etl.type_id,
            etl.genre_id
        FROM {{schema}}.event_type_link etl
        WHERE etl.event_id = e.id
    ) event_type
) et_data ON TRUE

-- Main image
LEFT JOIN LATERAL (
    SELECT
        pil.pluto_image_id AS image_id,
        format('{{base_api_url}}/api/image/%s', pil.pluto_image_id) AS image_url
    FROM {{schema}}.pluto_image_link pil
    WHERE pil.context = 'event'
      AND pil.context_id = e.id
      AND pil.identifier = 'main'
    ORDER BY pil.id
    LIMIT 1
) main_image_link ON TRUE

-- Permissions
LEFT JOIN {{schema}}.user_event_link uel
    ON uel.event_id = e.id
    AND uel.user_id = $3

LEFT JOIN {{schema}}.user_organization_link uol
    ON uol.organization_id = e.organization_id
    AND uol.user_id = $3

LEFT JOIN {{schema}}.user_venue_link uvl
    ON uvl.venue_id = e.venue_id
    AND uvl.user_id = $3

WHERE eo.id = $1
AND (
    edt.start_date >= $2::date
    OR edt.start_date IS NULL
)

ORDER BY
    edt.start_date NULLS LAST,
    edt.start_time NULLS LAST