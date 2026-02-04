INSERT INTO {{schema}}.venue (
    organization_id,
    name,
    description,
    opened_at,
    closed_at,
    contact_email,
    contact_phone,
    website_link,
    street,
    house_number,
    postal_code,
    city,
    country,
    state,
    wkb_pos,
    created_by
)
VALUES (
    $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,
    ST_SetSRID(ST_MakePoint($15,$16),4326),
    $17
)
RETURNING id
