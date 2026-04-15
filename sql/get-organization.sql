SELECT
    uuid::text,
    name,
    description,
    contact_email,
    contact_phone,
    web_link,
    street,
    house_number,
    address_addition,
    postal_code,
    city,
    country,
    holding_org_uuid,
    legal_form,
    nonprofit,
    address_addition,
    state,
    ST_X(point) AS lon,
    ST_Y(point) AS lat
FROM {{schema}}.organization
WHERE uuid = $1