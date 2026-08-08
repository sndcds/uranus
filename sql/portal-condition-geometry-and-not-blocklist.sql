AND (
    p.wkb_geometry IS NULL
    OR ST_Covers(
        p.wkb_geometry,
        COALESCE(edp.venue_point, ep.venue_point)
    )
)
AND NOT EXISTS (
    SELECT 1
    FROM {{schema}}.portal_org_blocklist b
    WHERE b.portal_uuid = p.uuid
        AND b.org_uuid = ep.org_uuid
)