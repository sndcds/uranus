SELECT
    name
FROM {{schema}}.organization
WHERE id = $1