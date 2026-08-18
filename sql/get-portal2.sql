SELECT
    uuid,
    slug,
    org_uuid,
    name,
    description,
    geometry_mode,
    ST_AsGeoJSON(geometry)::json AS geometry,
    filter,
    filter_type,

    CASE
        WHEN pil_web_logo.pluto_image_uuid IS NOT NULL
            THEN format('{{base_api_url}}/api/image/%s', pil_web_logo.pluto_image_uuid)
        END AS web_logo_url,

    CASE
        WHEN pil_main_image.pluto_image_uuid IS NOT NULL
            THEN format('{{base_api_url}}/api/image/%s', pil_main_image.pluto_image_uuid)
        END AS main_image,

    CASE
        WHEN pil_background.pluto_image_uuid IS NOT NULL
            THEN format('{{base_api_url}}/api/image/%s', pil_background.pluto_image_uuid)
        END AS background_image_url,

    CASE
        WHEN pil_footer_logo.pluto_image_uuid IS NOT NULL
            THEN format('{{base_api_url}}/api/image/%s', pil_footer_logo.pluto_image_uuid)
        END AS footer_logo_url,

    config

FROM {{schema}}.portal2 p

LEFT JOIN {{schema}}.pluto_image_link pil_web_logo
    ON pil_web_logo.context = 'portal'
        AND pil_web_logo.context_uuid = p.uuid
        AND pil_web_logo.identifier = 'web_logo'

LEFT JOIN {{schema}}.pluto_image_link pil_main_image
    ON pil_web_logo.context = 'portal'
        AND pil_web_logo.context_uuid = p.uuid
        AND pil_web_logo.identifier = 'main_image'

LEFT JOIN {{schema}}.pluto_image_link pil_background
    ON pil_background.context = 'portal'
        AND pil_background.context_uuid = p.uuid
        AND pil_background.identifier = 'background_image'

LEFT JOIN {{schema}}.pluto_image_link pil_footer_logo
    ON pil_footer_logo.context = 'portal'
        AND pil_footer_logo.context_uuid = p.uuid
        AND pil_footer_logo.identifier = 'footer_logo'

{{condition}}