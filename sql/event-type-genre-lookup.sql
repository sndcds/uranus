WITH types_with_genres AS (
    SELECT
        et.iso_639_1 AS lang,
        et.type_id,
        et.name AS type_name,
        jsonb_object_agg(
            gt.genre_id,
            gt.name
        ) FILTER (WHERE gt.genre_id IS NOT NULL) AS genres
    FROM {{schema}}.event_type et
    LEFT JOIN {{schema}}.genre_type gt
    ON gt.type_id = et.type_id
    AND gt.iso_639_1 = et.iso_639_1
    WHERE et.iso_639_1 = ANY(ARRAY['da','de','en']) -- $1 = array of locales, e.g. ['da','de','en']
    GROUP BY et.iso_639_1, et.type_id, et.name
)
SELECT
    lang,
    jsonb_object_agg(
        type_id,
        jsonb_build_object(
            'name', type_name,
            'genres', COALESCE(genres, '{}'::jsonb)
        )
    ) AS types
FROM types_with_genres
GROUP BY lang