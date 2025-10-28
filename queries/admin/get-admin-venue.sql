SELECT
    name,
    description,
    organizer_id,
    opened_at,
    closed_at,
    contact_email,
    contact_phone,
    website_url,
    street,
    house_number,
    postal_code,
    city,
    state_code,
    country_code,
    ST_X(venue.wkb_geometry) AS lon,
    ST_Y(venue.wkb_geometry) AS lat
FROM {{schema}}.venue
WHERE id = $1
