SELECT
    COALESCE(
        (
            SELECT uvl.permissions
            FROM {{schema}}.user_venue_link uvl
            WHERE uvl.user_id = $1
            AND uvl.venue_id = $2
            LIMIT 1
        ), 0
    )
    |
    COALESCE(
        (
            SELECT uol.permissions
            FROM {{schema}}.user_organization_link uol
            JOIN {{schema}}.venue v ON v.organization_id = uol.organization_id
            WHERE uol.user_id = $1
            AND v.id = $2
            LIMIT 1
        ), 0
    ) AS permissions