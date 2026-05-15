SELECT
    p.org_uuid,
    p.name,
    p.description,
    p.spatial_filter_mode,
    p.prefilter,
    ST_AsGeoJSON(wkb_geometry)::json AS geometry,
    p.style,
    p.header,
    p.footer

FROM {{schema}}.portal p

JOIN {{schema}}.user_organization_link uol
    ON uol.org_uuid = p.org_uuid
    AND uol.user_uuid = $2::uuid

WHERE p.uuid = $1::uuid