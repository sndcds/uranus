SELECT
    s.id AS space_id,
    s.name AS space_name,
    v.organizer_id AS organizer_id
FROM
    {{schema}}.space AS s
        JOIN
    {{schema}}.venue AS v ON s.venue_id = v.id
WHERE
    s.venue_id = $1
ORDER BY
    s.name;