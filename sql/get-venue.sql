SELECT
    v.id,
    v.name,
    v.type,
    vti.name AS type_name,
    vti.description AS type_description,
    TO_CHAR(v.opened_at, 'YYYY-MM-DD') AS opened_at,
    TO_CHAR(v.closed_at, 'YYYY-MM-DD') AS closed_at,
    v.description,
    v.street,
    v.house_number,
    v.postal_code,
    v.city,
    v.country,
    v.state,
    v.contact_email,
    v.contact_phone,
    v.website_link,
    ST_X(v.geo_pos) AS lon,
    ST_Y(v.geo_pos) AS lat,
    v.organization_id,

    o.name AS org_name,
    o.website_link AS org_website_link,
    o.city AS org_city,
    o.country AS org_country,

    COALESCE(
            json_agg(
                    json_build_object(
                            'id', s.id,
                            'name', s.name,
                            'total_capacity', s.total_capacity,
                            'seating_capacity', s.seating_capacity,
                            'building_level', s.building_level,
                            'website_link', s.website_link,
                            'description', s.description,
                            'area_sqm', s.area_sqm,
                            'space_type', s.space_type,
                            'space_type_name', sti.name,
                            'space_type_description', sti.description
                    )
            ) FILTER (WHERE s.id IS NOT NULL),
            '[]'
    ) AS spaces

FROM {{schema}}.venue v
LEFT JOIN {{schema}}.organization o ON o.id = v.organization_id
LEFT JOIN {{schema}}.space s ON s.venue_id = v.id

LEFT JOIN {{schema}}.venue_type vt ON vt.key = v.type
LEFT JOIN {{schema}}.venue_type_i18n vti
ON vti.key = vt.key
AND vti.iso_639_1 = $2

LEFT JOIN {{schema}}.space_type st ON st.key = s.space_type
LEFT JOIN {{schema}}.space_type_i18n sti
ON sti.key = st.key
AND sti.iso_639_1 = $2

WHERE v.id = $1
GROUP BY
    v.id,
    o.id,
    vti.name,
    vti.description