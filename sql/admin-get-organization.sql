SELECT
    o.name,
    o.description,
    o.legal_form_id,
    o.holding_organization_id,
    o.nonprofit,
    o.contact_email,
    o.contact_phone,
    o.website_url,
    o.street,
    o.house_number,
    o.postal_code,
    o.city,
    o.state_code,
    o.country_code,
    o.address_addition,
    ST_X(o.wkb_pos) AS lon,
    ST_Y(o.wkb_pos) AS lat
FROM {{schema}}.organization o
JOIN {{schema}}.user_organization_link uol ON uol.organization_id = o.id AND uol.user_id = $2
WHERE o.id = $1