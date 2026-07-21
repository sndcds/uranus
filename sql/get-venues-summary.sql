WITH filtered_venues AS (
    SELECT
        v.type
    FROM {{schema}}.venue v
    WHERE TRUE
        {{conditions}}
)

SELECT jsonb_build_object(
    'total_venue_count',
    (SELECT COUNT(*) FROM filtered_venues),
    'type_summary',
    (
        SELECT COALESCE(
            jsonb_agg(
                jsonb_build_object(
                    'type', type,
                    'venue_count', venue_count
                )
                ORDER BY venue_count DESC
            ),
            '[]'::jsonb
        )
        FROM (
            SELECT
                type,
                COUNT(*) AS venue_count
                FROM filtered_venues
                GROUP BY type
        ) s
    )
) AS summary