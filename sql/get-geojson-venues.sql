SELECT
    v.name AS venue_name,
    v.city AS venue_city,
    ST_X(v.wkb_pos) AS venue_lon,
    ST_Y(v.wkb_pos) AS venue_lat,
    v.website_link AS venue_link,
    v.type AS venue_type_key,
    vti.name AS venue_type_name
FROM {{schema}}.venue v
LEFT JOIN {{schema}}.venue_type_i18n vti ON vti.key = v.type AND vti.iso_639_1 = $1
GROUP BY v.id, v.name, v.city, v.wkb_pos, vti.name