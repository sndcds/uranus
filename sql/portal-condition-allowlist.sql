AND EXISTS (
    SELECT 1
    FROM {{schema}}.portal_org_allowlist a
    WHERE a.portal_uuid = p.uuid
        AND a.allowed_org_uuid = ep.org_uuid
)