SELECT
    v.uuid::text,
    v.type,
    v.name,
    v.street,
    v.house_number,
    v.city,
    v.country,
    v.scope,
    vt.marker_style,
    ST_AsGeoJSON(v.point) AS point,
    ST_AsGeoJSON(v.building) AS building,
    v.web_link,
    CASE
        WHEN pil.pluto_image_uuid IS NOT NULL
            THEN format(
                '{{base_api_url}}/api/image/%s',
                pil.pluto_image_uuid::text
                 )
        ELSE NULL
        END AS logo_url

FROM {{schema}}.venue v

LEFT JOIN {{schema}}.pluto_image_link pil
    ON pil.context = 'venue'
        AND pil.context_uuid = v.uuid
        AND pil.identifier = 'main_logo'

LEFT JOIN {{schema}}.venue_type vt
    ON vt.key = v.type

JOIN {{schema}}.portal2 p
    ON p.uuid = $5::uuid

LEFT JOIN {{schema}}.portal_geometry pg
    ON (
        (
            $7::uuid IS NULL
                AND pg.portal_uuid = p.uuid
                AND pg.parent_uuid IS NULL
        )
        OR
        (
            $7::uuid IS NOT NULL
                AND pg.uuid = $7::uuid
        )
    )

WHERE
    -- API bounding box
    v.point IS NOT NULL
    AND ST_Covers(
        ST_MakeEnvelope($1, $2, $3, $4, 4326),
        v.point
    )

    -- Geographic portal filter
    AND (
        p.geographic_filter = 'none'
        OR (
            p.geographic_filter = 'geometry'
            AND pg.uuid IS NOT NULL
            AND ST_Covers(pg.geometry, v.point)
        )
    )

    -- Organization portal filter
    AND (
        p.org_filter = 'all'
        OR (
            p.org_filter = 'allowlist'
            AND EXISTS (
                SELECT 1
                FROM {{schema}}.portal_org_allowlist a
                WHERE a.portal_uuid = p.uuid
                    AND a.org_uuid = v.org_uuid
            )
        )
        OR (
            p.org_filter = 'blocklist'
            AND NOT EXISTS (
                SELECT 1
                FROM {{schema}}.portal_org_blocklist b
                WHERE b.portal_uuid = p.uuid
                    AND b.org_uuid = v.org_uuid
            )
        )
    )

    -- Scope
    AND (
        cardinality($6::text[]) = 0
        OR v.scope = ANY($6::text[])
    )
