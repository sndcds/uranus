AND (
    -- No geographic restriction
    p.geographic_filter = 'none'

    OR (
        -- Geometry restriction
        p.geographic_filter = 'geometry'
        AND pg.uuid IS NOT NULL
        AND ST_Covers(
            pg.geometry,
            COALESCE(edp.venue_point, ep.venue_point)
        )
    )
)

AND (
    -- No organization filter
    p.org_filter = 'all'

    OR (
        -- Allowlist
        p.org_filter = 'allowlist'
        AND EXISTS (
            SELECT 1
            FROM {{schema}}.portal_org_allowlist a
            WHERE a.portal_uuid = p.uuid
                AND a.org_uuid = ep.org_uuid
        )
    )

    OR (
        -- Blocklist
        p.org_filter = 'blocklist'
        AND NOT EXISTS (
            SELECT 1
            FROM {{schema}}.portal_org_blocklist b
            WHERE b.portal_uuid = p.uuid
                AND b.org_uuid = ep.org_uuid
        )
    )
)