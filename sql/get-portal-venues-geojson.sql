SELECT
    v.uuid::text,
    v.name,
    v.city,
    v.country,
    ST_X(v.point) AS lon,
    ST_Y(v.point) AS lat

FROM {{schema}}.venue v

LEFT JOIN {{schema}}.portal p
    ON p.uuid = $1::uuid

WHERE
	(
	    (
	        v.point IS NOT NULL
                AND ST_Within(v.point, ST_MakeEnvelope($4, $5, $6, $7, 4326))
                AND ST_Within(v.point, p.wkb_geometry)
	    )
        OR v.uuid = ANY($2::uuid[]) -- Whitelist
	)
	AND NOT (
        v.uuid = ANY($3::uuid[])    -- Blacklist
    )