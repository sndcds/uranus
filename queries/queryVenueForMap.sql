SELECT
    venue.name AS venue_name,
    venue.city AS venue_city,
    ST_X(venue.wkb_geometry) AS venue_lon,
    ST_Y(venue.wkb_geometry) AS venue_lat,
    STRING_AGG(venue_type.name, ', ') AS venue_type_list,
    MAX(venue_url.url) AS venue_url
FROM {{schema}}.venue AS venue
LEFT JOIN {{schema}}.venue_type_links AS venue_type_links
    ON venue_type_links.venue_id = venue.id
LEFT JOIN {{schema}}.venue_type AS venue_type
    ON venue_type_links.venue_type_id = venue_type.type_id
    AND venue_type.iso_639_1 = $1
LEFT JOIN {{schema}}.venue_url AS venue_url
    ON venue_url.venue_id = venue.id
    AND venue_url.link_type = 'website'
GROUP BY venue.id, venue.name, venue.city, venue.wkb_geometry;