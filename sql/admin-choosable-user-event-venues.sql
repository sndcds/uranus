WITH allowed_spaces AS (
    SELECT
        v.uuid AS venue_uuid,
        v.name AS venue_name,
        s.uuid AS space_uuid,
        s.name AS space_name,
        v.city AS city,
        v.country AS country
    FROM {{schema}}.space s
    JOIN {{schema}}.venue v ON v.uuid = s.venue_uuid
    JOIN {{schema}}.user_organization_link uol ON uol.org_uuid = v.org_uuid
    WHERE uol.user_uuid = $1::uuid
    AND (uol.permissions & 0x00000800) <> 0

    UNION

    SELECT
        v.uuid AS venue_uuid,
        v.name AS venue_name,
        s.uuid AS space_uuid,
        s.name AS space_name,
        v.city AS city,
        v.country AS country
    FROM {{schema}}.space s
    JOIN {{schema}}.user_venue_link uvl ON uvl.venue_uuid = s.venue_uuid
    JOIN {{schema}}.venue v ON v.uuid = s.venue_uuid
    WHERE uvl.user_uuid = $1::uuid
    AND (uvl.permissions & 0x00000800) <> 0

    UNION

    SELECT
        v.uuid AS venue_uuid,
        v.name AS venue_name,
        s.uuid AS space_uuid,
        s.name AS space_name,
        v.city AS city,
        v.country AS country
    FROM {{schema}}.space s
    JOIN {{schema}}.user_space_link usl ON usl.space_uuid = s.uuid
    JOIN {{schema}}.venue v ON v.uuid = s.venue_uuid
    WHERE usl.user_uuid = $1::uuid
    AND usl.permissions <> 0
),
allowed_venues AS (
    -- Venues user can select directly, even without spaces
    SELECT
        v.uuid AS venue_uuid,
        v.name AS venue_name,
        NULL::uuid AS space_uuid,
        NULL::text AS space_name,
        v.city AS city,
        v.country AS country
    FROM {{schema}}.venue v
    -- From organization permission
    JOIN {{schema}}.user_organization_link uol ON uol.org_uuid = v.org_uuid
    WHERE uol.user_uuid = $1::uuid
    AND (uol.permissions & 0x00000800) <> 0

    UNION

    -- From direct venue permission
    SELECT
        v.uuid AS venue_uuid,
        v.name AS venue_name,
        NULL::uuid AS space_uuid,
        NULL::text AS space_name,
        v.city AS city,
        v.country AS country
    FROM {{schema}}.venue v
    JOIN {{schema}}.user_venue_link uvl ON uvl.venue_uuid = v.uuid
    WHERE uvl.user_uuid = $1::uuid
    AND (uvl.permissions & 0x00000800) <> 0
)
SELECT * FROM allowed_spaces
UNION
SELECT * FROM allowed_venues
ORDER BY venue_name, space_name