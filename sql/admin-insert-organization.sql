INSERT INTO {{schema}}.organization (
    name,
    description,
    legal_form_id,
    nonprofit,
    contact_email,
    contact_phone,
    website_url,
    street,
    house_number,
    postal_code,
    city,
    country,
    state,
    address_addition,
    wkb_pos,
    created_by
)
VALUES (
    $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,
    ST_SetSRID(ST_MakePoint($15,$16),4326),
    $17
)
RETURNING id