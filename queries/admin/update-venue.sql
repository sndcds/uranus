UPDATE {{schema}}.venue
SET
    name = $2,
    description = $3,
    opened_at = $4,
    closed_at = $5,
    contact_email = $6,
    contact_phone = $7,
    website_url = $8,
    street = $9,
    house_number = $10,
    postal_code = $11,
    city = $12,
    state_code = $13,
    country_code = $14,
    wkb_geometry = ST_SetSRID(ST_MakePoint($15, $16), 4326)
WHERE id = $1;
