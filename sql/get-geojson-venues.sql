WITH normalized AS (
    SELECT
        edp.event_date_uuid,
        edp.event_uuid,

        -- Venue inheritance
        COALESCE(edp.venue_uuid, ep.venue_uuid) AS venue_uuid,

        -- Status inheritance
        CASE
            WHEN edp.release_status = 'inherited'
                THEN ep.release_status
            ELSE edp.release_status
            END AS effective_status

    FROM {{schema}}.event_date_projection edp
    JOIN {{schema}}.event_projection ep
    ON ep.event_uuid = edp.event_uuid
    WHERE edp.start_date >= $1
)
SELECT
    venue_uuid::text AS uuid,
    v.name AS name,
    v.city AS city,
    ST_X(v.point) AS lon,
    ST_Y(v.point) AS lat,
    COUNT(*) AS count
FROM normalized
LEFT JOIN {{schema}}.venue v ON v.uuid = venue_uuid
WHERE effective_status NOT IN ('draft', 'review')
GROUP BY venue_uuid, v.point, v.name, v.city