AND NOT EXISTS (
    SELECT 1
    FROM {{uranus}}.portal_org_blocklist b
    WHERE b.portal_uuid = p.uuid
        AND b.blocked_org_uuid = ep.org_uuid
)