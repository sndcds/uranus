INSERT INTO {{schema}}.venue (
    org_uuid,
    name,
    description,
    opened_at,
    closed_at,
    contact_email,
    contact_phone,
    web_link,
    street,
    house_number,
    postal_code,
    city,
    country,
    state,
    point,
    created_by
)
VALUES (
    $1::uuid, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14,
    ST_SetSRID(ST_MakePoint($15, $16),4326),
    $17::uuid
)
RETURNING id
