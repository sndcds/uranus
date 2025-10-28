SELECT
    name,
    description,
    legal_form_id,
    holding_organizer_id,
    nonprofit,
    contact_email,
    contact_phone,
    website_url,
    street,
    house_number,
    postal_code,
    city,
    state_code,
    country_code,
    address_addition,
    ST_X(organizer.wkb_geometry) AS lon,
    ST_Y(organizer.wkb_geometry) AS lat
FROM {{schema}}.organizer
WHERE id = $1

