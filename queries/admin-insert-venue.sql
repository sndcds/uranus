INSERT INTO {{schema}}.venue (
    name,
    description,
    opened_at,
    closed_at,
    contact_email,
    contact_phone,
    website_url,
    street,
    house_number,
    postal_code,
    city,
    country_code,
    state_code,
    wkb_pos,
    created_by
)
VALUES (
    $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,
    ST_SetSRID(ST_MakePoint($14,$15),4326),
    $16
)
RETURNING id
