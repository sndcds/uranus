UPDATE {{schema}}.venue
SET
    name = $2,
    description = $3,
    opened_at = $4,
    closed_at = $5,
    contact_email = $6,
    contact_phone = $7,
    website_link = $8,
    street = $9,
    house_number = $10,
    postal_code = $11,
    city = $12,
    country = $13,
    state = $14,
    geo_pos = ST_SetSRID(ST_MakePoint($15, $16), 4326),
    modified_by = $17
WHERE id = $1
