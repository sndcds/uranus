AND (
    p.wkb_geometry IS NULL
    OR ST_Covers(
        p.wkb_geometry,
        COALESCE(edp.venue_point, ep.venue_point)
    )
)
AND EXISTS (
    SELECT 1
    FROM {{schema}}.portal_org_allowlist a
    WHERE a.portal_uuid = p.uuid
        AND a.allowed_org_uuid = ep.org_uuid
)