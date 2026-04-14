SELECT
    u.uuid,
    u.display_name,
    u.first_name,
    u.last_name,
    o.name
FROM {{schema}}.user u
JOIN {{schema}}.organization o ON o.uuid = $1::uuid
WHERE u.email = $2