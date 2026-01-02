WITH allowed_spaces AS (
    -- A. From organization permission (all spaces)
    SELECT
        v.id AS venue_id,
        v.name AS venue_name,
        s.id AS space_id,
        s.name AS space_name,
        v.city AS city,
        v.country_code AS country_code
    FROM {{schema}}.space s
    JOIN {{schema}}.venue v ON v.id = s.venue_id
    JOIN {{schema}}.user_organization_link uol ON uol.organization_id = v.organization_id
    WHERE uol.user_id = $1
    AND (uol.permissions & 0x00000800) <> 0

    UNION

    -- B. From venue permission (all spaces)
    SELECT
        v.id AS venue_id,
        v.name AS venue_name,
        s.id AS space_id,
        s.name AS space_name,
        v.city AS city,
        v.country_code AS country_code
    FROM {{schema}}.space s
    JOIN {{schema}}.user_venue_link uvl ON uvl.venue_id = s.venue_id
    JOIN {{schema}}.venue v ON v.id = s.venue_id
    WHERE uvl.user_id = $1
    AND (uvl.permissions & 0x00000800) <> 0

    UNION

    -- C. From direct space permissions
    SELECT
        v.id AS venue_id,
        v.name AS venue_name,
        s.id AS space_id,
        s.name AS space_name,
        v.city AS city,
        v.country_code AS country_code
    FROM {{schema}}.space s
    JOIN {{schema}}.user_space_link usl ON usl.space_id = s.id
    JOIN {{schema}}.venue v ON v.id = s.venue_id
    WHERE usl.user_id = $1
    AND usl.permissions <> 0
),
allowed_venues AS (
    -- Venues user can select directly, even without spaces
    SELECT
        v.id AS venue_id,
        v.name AS venue_name,
        NULL::integer AS space_id,
        NULL::text AS space_name,
        v.city AS city,
        v.country_code AS country_code
    FROM {{schema}}.venue v
    -- From organization permission
    JOIN {{schema}}.user_organization_link uol ON uol.organization_id = v.organization_id
    WHERE uol.user_id = $1
    AND (uol.permissions & 0x00000800) <> 0

    UNION

    -- From direct venue permission
    SELECT
        v.id AS venue_id,
        v.name AS venue_name,
        NULL::integer AS space_id,
        NULL::text AS space_name,
        v.city AS city,
        v.country_code AS country_code
    FROM {{schema}}.venue v
    JOIN {{schema}}.user_venue_link uvl ON uvl.venue_id = v.id
    WHERE uvl.user_id = $1
    AND (uvl.permissions & 0x00000800) <> 0
)
SELECT * FROM allowed_spaces
UNION
SELECT * FROM allowed_venues
ORDER BY venue_name, space_name