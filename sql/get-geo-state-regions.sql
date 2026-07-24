SELECT
    r.code,
    r.name,
    r.slug
FROM {{schema}}.geolist_region r
JOIN {{schema}}.geolist_state s
    ON s.country_code = r.country_code
        AND s.code = r.state_code
JOIN {{schema}}.geolist_country c
    ON c.code = s.country_code
WHERE c.slug = $1
    AND s.slug = $2
ORDER BY r.name