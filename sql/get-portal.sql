SELECT
    uuid,
    name,
    description,
    org_uuid,
    spatial_filter_mode,
    prefilter,
    ST_AsGeoJSON(wkb_geometry)::json AS geometry,
    style
FROM {{schema}}.portal
WHERE uuid = $1::uuid