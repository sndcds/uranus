SELECT
    uuid,
    name,
    description,
    org_uuid,
    spatial_filter_mode,
    prefilter,
    ST_AsGeoJSON(wkb_geometry)::json AS geometry,
    style,
    pil_web_logo.pluto_image_uuid AS web_logo_image_uuid,
    pil_background.pluto_image_uuid AS background_image_uuid

FROM {{schema}}.portal p

LEFT JOIN {{schema}}.pluto_image_link pil_web_logo
    ON pil_web_logo.context = 'portal'
    AND pil_web_logo.context_uuid = p.uuid
    AND pil_web_logo.identifier = 'web_logo'

LEFT JOIN {{schema}}.pluto_image_link pil_background
    ON pil_background.context = 'portal'
    AND pil_background.context_uuid = p.uuid
    AND pil_background.identifier = 'background_image'

WHERE uuid = $1::uuid
