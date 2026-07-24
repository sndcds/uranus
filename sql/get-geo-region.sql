SELECT
    c.code,
    c.name,
    c.slug,
    s.code,
    s.name,
    s.slug,
    r.code,
    r.name,
    r.slug,
    ST_AsGeoJSON(r.wkb_geometry)

FROM {{schema}}.geolist_country c

JOIN {{schema}}.geolist_state s
    ON s.country_code = c.code

JOIN {{schema}}.geolist_region r
    ON r.country_code = c.code
        AND r.state_code = s.code

WHERE c.slug = $1
    AND s.slug = $2
    AND r.slug = $3

LIMIT 1