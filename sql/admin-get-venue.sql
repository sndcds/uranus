SELECT
    v.name AS name,
    v.description,
    v.organization_id,
    v.opened_at,
    v.closed_at,
    v.contact_email,
    v.contact_phone,
    v.website_url,
    v.street,
    v.house_number,
    v.postal_code,
    v.city,
    v.state_code,
    v.country_code,
    ST_X(v.wkb_pos) AS lon,
    ST_Y(v.wkb_pos) AS lat
FROM {{schema}}.venue v
JOIN {{schema}}.organization o ON o.id = v.organization_id
JOIN {{schema}}.user_organization_link uol ON uol.organization_id = o.id AND uol.user_id = $2
WHERE v.id = $1
