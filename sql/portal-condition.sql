AND (
    -- geometry is not required
    p.filter_type IN ('allowlist', 'blocklist')

    OR
    -- geometry is required and matches
    (
        p.filter_type IN (
            'geometry',
            'geometry_and_allowlist',
            'geometry_and_blocklist'
        )
        AND (
            p.geometry IS NULL
            OR ST_Covers(
                p.geometry,
                COALESCE(edp.venue_point, ep.venue_point)
            )
        )
    )
)
AND (
    -- no organization filter
    p.filter_type = 'geometry'

    OR
    -- allowlist
    (
        p.filter_type IN ('allowlist', 'geometry_and_allowlist')
        AND EXISTS (
            SELECT 1
            FROM {{schema}}.portal_org_allowlist a
            WHERE a.portal_uuid = p.uuid
                AND a.org_uuid = ep.org_uuid
        )
    )

    OR
    -- blocklist
    (
        p.filter_type IN ('blocklist', 'geometry_and_blocklist')
        AND NOT EXISTS (
            SELECT 1
            FROM {{schema}}.portal_org_blocklist b
            WHERE b.portal_uuid = p.uuid
                AND b.org_uuid = ep.org_uuid
        )
    )
)