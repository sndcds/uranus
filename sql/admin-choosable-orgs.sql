WITH user_organization_access AS (
    SELECT DISTINCT
        o.uuid AS uuid,
        o.name AS name
    FROM {{schema}}.organization o
    JOIN {{schema}}.user_organization_link uol
    ON uol.org_uuid = o.uuid
    WHERE uol.user_uuid = $1
),
user_venue_access AS (
    SELECT DISTINCT
        o.uuid AS uuid,
        o.name AS name
    FROM {{schema}}.venue v
    JOIN {{schema}}.organization o ON o.uuid = v.org_uuid
    JOIN {{schema}}.user_venue_link uvl ON uvl.venue_uuid = v.uuid
    WHERE uvl.user_uuid = $1
),
accessible_organizations AS (
    SELECT * FROM user_organization_access
    UNION
    SELECT * FROM user_venue_access
)
SELECT
    uuid,
    name
FROM accessible_organizations
ORDER BY LOWER(name)