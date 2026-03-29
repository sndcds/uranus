SELECT
    COALESCE(
        (
            SELECT uvl.permissions
            FROM {{schema}}.user_venue_link uvl
            WHERE uvl.user_uuid = $1
            AND uvl.venue_uuid = $2
            LIMIT 1
        ), 0
    )
    |
    COALESCE(
        (
            SELECT uol.permissions
            FROM {{schema}}.user_organization_link uol
            JOIN {{schema}}.venue v ON v.org_uuid = uol.org_uuid
            WHERE uol.user_uuid = $1
            AND v.uuid = $2
            LIMIT 1
        ), 0
    ) AS permissions