SELECT
    venue.name AS venue_name,
    venue.city AS venue_city,
    ST_X(venue.wkb_pos) AS venue_lon,
    ST_Y(venue.wkb_pos) AS venue_lat,
    STRING_AGG(venue_type.name, ', ') AS venue_type_list,
    MAX(venue_link.url) AS venue_link
FROM {{schema}}.venue AS venue
LEFT JOIN {{schema}}.venue_type_link AS venue_type_link
    ON venue_type_link.venue_id = venue.id
LEFT JOIN {{schema}}.venue_type AS venue_type
    ON venue_type_link.venue_type_id = venue_type.type_id
    AND venue_type.iso_639_1 = $1
LEFT JOIN {{schema}}.venue_link AS venue_link
    ON venue_link.venue_id = venue.id
    AND venue_link.link_type = 'website'
GROUP BY venue.id, venue.name, venue.city, venue.wkb_pos