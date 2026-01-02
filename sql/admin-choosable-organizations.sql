WITH user_org_access AS (
    SELECT DISTINCT
        o.id AS id,
        o.name AS name
    FROM {{schema}}.organization o
    JOIN {{schema}}.user_organization_link uol
    ON uol.organization_id = o.id
    WHERE uol.user_id = $1
),
user_venue_access AS (
    SELECT DISTINCT
        o.id AS id,
        o.name AS name
    FROM {{schema}}.venue v
    JOIN {{schema}}.organization o ON o.id = v.organization_id
    JOIN {{schema}}.user_venue_link uvl ON uvl.venue_id = v.id
    WHERE uvl.user_id = $1
),
accessible_organizations AS (
    SELECT * FROM user_org_access
    UNION
    SELECT * FROM user_venue_access
)
SELECT
    id,
    name
FROM accessible_organizations
ORDER BY LOWER(name)