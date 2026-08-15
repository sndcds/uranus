SELECT
    v.uuid::text,
    v.type,
    v.name,
    v.street,
    v.house_number,
    v.city,
    v.country,
    v.scope,
    ST_X(v.point) AS lon,
    ST_Y(v.point) AS lat
FROM {{schema}}.venue v
JOIN {{schema}}.portal2 p
    ON p.uuid = $5::uuid
WHERE
    -- API bounding box
    v.point IS NOT NULL
    AND ST_Covers(
        ST_MakeEnvelope($1, $2, $3, $4, 4326),
        v.point
    )

    -- Portal filter
    AND (
        CASE p.filter_type
            WHEN 'geometry' THEN
                p.geometry IS NOT NULL
                AND ST_Covers(p.geometry, v.point)

            WHEN 'allowlist' THEN
                EXISTS (
                    SELECT 1
                    FROM {{schema}}.portal_org_allowlist a
                    WHERE a.portal_uuid = p.uuid
                        AND a.org_uuid = v.org_uuid
                )

            WHEN 'blocklist' THEN
                NOT EXISTS (
                    SELECT 1
                    FROM {{schema}}.portal_org_blocklist b
                    WHERE b.portal_uuid = p.uuid
                    AND b.org_uuid = v.org_uuid
                )

            WHEN 'geometry_and_allowlist' THEN
                (
                    (
                        p.geometry IS NOT NULL
                            AND ST_Covers(p.geometry, v.point)
                    )
                    OR EXISTS (
                        SELECT 1
                        FROM {{schema}}.portal_org_allowlist a
                        WHERE a.portal_uuid = p.uuid
                        AND a.org_uuid = v.org_uuid
                    )
                )

            WHEN 'geometry_and_allowlist' THEN
                p.geometry IS NOT NULL
                AND ST_Covers(p.geometry, v.point)
                AND EXISTS (
                    SELECT 1
                    FROM {{schema}}.portal_org_allowlist a
                    WHERE a.portal_uuid = p.uuid
                    AND a.org_uuid = v.org_uuid
                )

            WHEN 'geometry_and_blocklist' THEN
                p.geometry IS NOT NULL
                AND ST_Covers(p.geometry, v.point)
                AND NOT EXISTS (
                    SELECT 1
                    FROM {{schema}}.portal_org_blocklist b
                    WHERE b.portal_uuid = p.uuid
                        AND b.org_uuid = v.org_uuid
                )
            ELSE FALSE
        END
    )

    -- Scope
    AND (
        cardinality($6::text[]) = 0
        OR v.scope = ANY($6::text[])
    )