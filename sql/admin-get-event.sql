SELECT
    e.id,
    e.external_id,
    e.source_url,
    e.release_status AS release_status,
    TO_CHAR(e.release_date, 'YYYY-MM-DD') AS release_date,
    e.content_iso_639_1 AS content_language,
    o.id AS organization_id,
    o.name AS organization_name,
    e.title,
    e.subtitle,
    e.description,
    e.summary,
    e.tags,
    e.occasion_type_id,
    v.id AS venue_id,
    v.name AS venue_name,
    v.street AS venue_street,
    v.house_number AS venue_house_number,
    v.postal_code AS venue_postal_code,
    v.city AS venue_city,
    v.country AS venue_country,
    v.state AS venue_state,
    ST_X(v.geo_pos) AS venue_lon,
    ST_Y(v.geo_pos) AS venue_lat,
    sd.id AS space_id,
    sd.name AS space_name,
    sd.total_capacity AS space_total_capacity,
    sd.seating_capacity AS space_seating_capacity,
    sd.building_level AS space_building_level,
    e.online_link,
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
    e.currency,
    cu.name AS currency_name,
    e.custom,
    e.style
FROM {{schema}}.event e
LEFT JOIN {{schema}}.organization o ON e.organization_id = o.id
LEFT JOIN {{schema}}.venue v ON v.id = e.venue_id
LEFT JOIN {{schema}}.currency cu ON cu.code = e.currency AND cu.iso_639_1 = $2
LEFT JOIN {{schema}}.user_organization_link uol
ON uol.organization_id = o.id
AND uol.user_id = $3
LEFT JOIN {{schema}}.user_event_link uel
ON uel.event_id = e.id
AND uel.user_id = $3
LEFT JOIN LATERAL (
    SELECT *
    FROM {{schema}}.space s2
    WHERE s2.id = e.space_id
    LIMIT 1
) sd ON TRUE
WHERE ((uol.permissions & $4) <> 0 OR (uel.permissions & $4) <> 0)
AND e.id = $1
LIMIT 1