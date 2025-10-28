UPDATE {{schema}}.organizer
SET
    name = $2,
    description = $3,
    legal_form_id = $4,
    holding_organizer_id = $5,
    nonprofit = $6,
    contact_email = $7,
    contact_phone = $8,
    website_url = $9,
    street = $10,
    house_number = $11,
    postal_code = $12,
    city = $13,
    state_code = $14,
    country_code = $15,
    address_addition = $16,
    wkb_geometry = ST_SetSRID(ST_MakePoint($17, $18), 4326)
WHERE id = $1;