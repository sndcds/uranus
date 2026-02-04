UPDATE {{schema}}.organization
SET
    name = $2,
    description = $3,
    legal_form_id = $4,
    nonprofit = $5,
    contact_email = $6,
    contact_phone = $7,
    website_link = $8,
    street = $9,
    house_number = $10,
    postal_code = $11,
    city = $12,
    country = $13,
    state = $14,
    address_addition = $15,
    wkb_pos = ST_SetSRID(ST_MakePoint($16, $17), 4326),
    modified_by = $18
WHERE id = $1