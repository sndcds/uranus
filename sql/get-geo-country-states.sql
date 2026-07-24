SELECT
    s.country_code,
    s.code AS code,
    s.name AS name,
    s.slug AS slug

FROM {{schema}}.geolist_state s

JOIN {{schema}}.geolist_country c
    ON c.code = s.country_code

WHERE c.slug = $1

ORDER BY s.name
