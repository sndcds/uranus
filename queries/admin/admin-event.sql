WITH
    event_types AS (
        SELECT type_id AS event_type_id
        FROM {{schema}}.event_type_links
        WHERE event_id = $1
    ),
    genre_pairs AS (
        SELECT
            gt.event_type_id,
            egl.type_id AS genre_type_id
        FROM {{schema}}.event_genre_links egl
        JOIN {{schema}}.genre_type gt ON gt.type_id = egl.type_id
        WHERE egl.event_id = $1
        GROUP BY gt.event_type_id, egl.type_id
    ),
    combined AS (
        SELECT
            et.event_type_id,
            gp.genre_type_id
        FROM event_types et
        LEFT JOIN genre_pairs gp ON gp.event_type_id = et.event_type_id
    )

SELECT
    o.name AS organizer_name,
    COALESCE(s.name) AS space_name,
    e.id AS event_id,
    e.organizer_id AS organizer_id,
    e.space_id AS space_id,
    v.id AS venue_id,
    e.title,
    e.subtitle,
    e.description,
    e.teaser_text,
    e.languages,
    e.min_age,
    e.max_age,
    e.participation_info,
    e.meeting_point,
    e.source_url,
    e.custom,
    e.style,
    e.release_date,

    COALESCE(
        (
            SELECT JSON_AGG(JSON_BUILD_ARRAY(event_type_id, genre_type_id))
            FROM combined
        ), '[]'
    ) AS event_type_genre_pairs

FROM {{schema}}.event e
JOIN {{schema}}.organizer o ON o.id = e.organizer_id
LEFT JOIN {{schema}}.space s ON s.id = e.space_id
LEFT JOIN {{schema}}.venue v ON v.id = s.venue_id
WHERE e.id = $1
GROUP BY
    o.name, s.name, e.id, e.organizer_id, e.space_id, v.id,
    e.title, e.subtitle, e.description, e.teaser_text, e.languages,
    e.min_age, e.max_age, e.participation_info, e.meeting_point,
    e.source_url, e.custom, e.style, e.release_date;