SELECT
    e.uuid,
    e.external_id,
    e.source_link,
    e.release_status AS release_status,
    TO_CHAR(e.release_date, 'YYYY-MM-DD') AS release_date,
    e.categories,
    e.content_iso_639_1 AS content_language,
    o.uuid AS org_uuid,
    o.name AS org_name,
    e.title,
    e.subtitle,
    e.description,
    e.summary,
    e.tags,
    e.occasion_type_id,
    v.uuid AS venue_uuid,
    v.name AS venue_name,
    v.street AS venue_street,
    v.house_number AS venue_house_number,
    v.postal_code AS venue_postal_code,
    v.city AS venue_city,
    v.country AS venue_country,
    v.state AS venue_state,
    ST_X(v.point) AS venue_lon,
    ST_Y(v.point) AS venue_lat,
    sd.uuid AS space_uuid,
    sd.name AS space_name,
    sd.total_capacity AS space_total_capacity,
    sd.seating_capacity AS space_seating_capacity,
    sd.building_level AS space_building_level,
    e.online_link,
    e.registration_link,
    e.registration_email,
    e.registration_phone,
    TO_CHAR(e.registration_deadline, 'YYYY-MM-DD') AS registration_deadline,
    e.meeting_point,
    e.languages,
    e.participation_info,
    e.min_age,
    e.max_age,
    e.max_attendees,
    e.price_type,
    e.min_price,
    e.max_price,
    e.ticket_flags,
    e.ticket_link,
    e.currency,
    cu.name AS currency_name,
    e.visitor_info_flags,
    e.custom,
    e.style
FROM {{schema}}.event e
LEFT JOIN {{schema}}.organization o ON e.org_uuid = o.uuid
LEFT JOIN {{schema}}.venue v ON v.uuid = e.venue_uuid
LEFT JOIN {{schema}}.currency cu ON cu.code = e.currency AND cu.iso_639_1 = $2
LEFT JOIN {{schema}}.user_organization_link uol
ON uol.org_uuid = o.uuid
AND uol.user_uuid = $3::uuid
LEFT JOIN {{schema}}.user_event_link uel
ON uel.event_uuid = e.uuid
AND uel.user_uuid = $3::uuid
LEFT JOIN LATERAL (
    SELECT *
    FROM {{schema}}.space s2
    WHERE s2.uuid = e.space_uuid
    LIMIT 1
) sd ON TRUE
WHERE ((uol.permissions & $4) <> 0 OR (uel.permissions & $4) <> 0)
AND e.uuid = $1::uuid
LIMIT 1