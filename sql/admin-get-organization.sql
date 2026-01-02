SELECT
    name,
    description,
    legal_form_id,
    holding_organization_id,
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
    ST_X(organization.wkb_pos) AS lon,
    ST_Y(organization.wkb_pos) AS lat
FROM {{schema}}.organization
WHERE id = $1