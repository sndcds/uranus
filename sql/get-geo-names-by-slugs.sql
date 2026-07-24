SELECT
    c.name AS country_name,
    s.name AS state_name,
    r.name AS region_name
FROM uranus.geolist_country c
LEFT JOIN uranus.geolist_state s
    ON s.slug = $2
LEFT JOIN uranus.geolist_region r
    ON r.slug = $3
WHERE c.slug = $1