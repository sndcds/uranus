INSERT INTO {{schema}}.event_projection (
    event_uuid, org_uuid, venue_uuid, space_uuid, release_status,
    title, subtitle, description, summary, image_uuid, image_ai_label, languages, tags, categories, types,
    source_link, online_link, occasion_type_id, max_attendees, min_age, max_age,
    participation_info, meeting_point, ticket_flags, ticket_link,
    price_type, currency, min_price, max_price, visitor_info_flags,
    external_id, custom, style,
    org_name, org_postal_code, org_city, org_country, org_contact_email, org_contact_phone, org_link,
    venue_name, venue_street, venue_house_number, venue_postal_code, venue_city,
    venue_country, venue_state, venue_point, venue_link,
    space_name, space_total_capacity, space_seating_capacity, space_type,
    space_building_level, space_link, space_accessibility_summary,
    space_accessibility_flags, space_description,
    created_at, modified_at
)
SELECT DISTINCT ON (e.uuid)
    e.uuid,
    e.org_uuid,
    e.venue_uuid,
    e.space_uuid,
    e.release_status,
    e.title,
    e.subtitle,
    e.description,
    e.summary,
    main_image.pluto_image_uuid AS image_uuid,
    main_image.ai_label AS image_ai_label,
    e.languages,
    e.tags,
    e.categories,
    COALESCE(
        (
            SELECT jsonb_agg(jsonb_build_array(type_id, genre_id))
            FROM {{schema}}.event_type_link etl WHERE etl.event_uuid = e.uuid
        ),
        '[]'::jsonb
    ),
    e.source_link,
    e.online_link,
    e.occasion_type_id,
    e.max_attendees,
    e.min_age,
    e.max_age,
    e.participation_info,
    e.meeting_point,
    e.ticket_flags,
    e.ticket_link,
    e.price_type,
    e.currency,
    e.min_price,
    e.max_price,
    e.visitor_info_flags,
    e.external_id,
    e.custom,
    e.style,
    o.name,
    o.postal_code,
    o.city,
    o.country,
    o.contact_email,
    o.contact_phone,
    o.web_link,
    v.name AS venue_name,
    v.street AS venue_street,
    v.house_number AS venue_house_number,
    v.postal_code AS venue_postal_code,
    v.city AS venue_city,
    v.country AS venue_country,
    v.state AS venue_state,
    v.point AS venue_point,
    v.web_link AS venue_web_link,
    s.name,
    s.total_capacity,
    s.seating_capacity,
    s.space_type,
    s.building_level,
    s.web_link,
    s.accessibility_summary,
    s.accessibility_flags,
    s.description,
    NOW(),
    NOW()

FROM {{schema}}.event e
    LEFT JOIN {{schema}}.organization o ON o.uuid = e.org_uuid
    LEFT JOIN {{schema}}.venue v ON v.uuid = e.venue_uuid
    LEFT JOIN {{schema}}.space s ON s.uuid = e.space_uuid
    JOIN {{schema}}.event_date ed ON ed.event_uuid = e.uuid

-- Fetch main image
LEFT JOIN LATERAL(
    SELECT
        pil.pluto_image_uuid,
        COALESCE(pi.ai_label, 'none') AS ai_label
    FROM {{schema}}.pluto_image_link pil
    LEFT JOIN {{schema}}.pluto_image pi
        ON pi.uuid = pil.pluto_image_uuid
    WHERE pil.context = 'event'
        AND pil.context_uuid = e.uuid
        AND pil.identifier = 'main'
    LIMIT 1
) main_image ON TRUE

WHERE e.uuid = ANY($1::uuid[])
--  AND ed.start_date >= CURRENT_DATE

ON CONFLICT (event_uuid) DO UPDATE SET
    org_uuid = EXCLUDED.org_uuid,
    venue_uuid = EXCLUDED.venue_uuid,
    space_uuid = EXCLUDED.space_uuid,
    release_status = EXCLUDED.release_status,
    title = EXCLUDED.title,
    subtitle = EXCLUDED.subtitle,
    description = EXCLUDED.description,
    summary = EXCLUDED.summary,
    image_uuid = EXCLUDED.image_uuid,
    image_ai_label = EXCLUDED.image_ai_label,
    languages = EXCLUDED.languages,
    tags = EXCLUDED.tags,
    categories = EXCLUDED.categories,
    types = EXCLUDED.types,
    source_link = EXCLUDED.source_link,
    online_link = EXCLUDED.online_link,
    occasion_type_id = EXCLUDED.occasion_type_id,
    max_attendees = EXCLUDED.max_attendees,
    min_age = EXCLUDED.min_age,
    max_age = EXCLUDED.max_age,
    participation_info = EXCLUDED.participation_info,
    meeting_point = EXCLUDED.meeting_point,
    ticket_flags = EXCLUDED.ticket_flags,
    ticket_link = EXCLUDED.ticket_link,
    price_type = EXCLUDED.price_type,
    currency = EXCLUDED.currency,
    min_price = EXCLUDED.min_price,
    max_price = EXCLUDED.max_price,
    visitor_info_flags = EXCLUDED.visitor_info_flags,
    external_id = EXCLUDED.external_id,
    custom = EXCLUDED.custom,
    style = EXCLUDED.style,
    org_name = EXCLUDED.org_name,
    org_postal_code = EXCLUDED.org_postal_code,
    org_city = EXCLUDED.org_city,
    org_country = EXCLUDED.org_country,
    org_contact_email = EXCLUDED.org_contact_email,
    org_contact_phone = EXCLUDED.org_contact_phone,
    org_link = EXCLUDED.org_link,
    venue_name = EXCLUDED.venue_name,
    venue_street = EXCLUDED.venue_street,
    venue_house_number = EXCLUDED.venue_house_number,
    venue_postal_code = EXCLUDED.venue_postal_code,
    venue_city = EXCLUDED.venue_city,
    venue_country = EXCLUDED.venue_country,
    venue_state = EXCLUDED.venue_state,
    venue_point = EXCLUDED.venue_point,
    venue_link = EXCLUDED.venue_link,
    space_name = EXCLUDED.space_name,
    space_total_capacity = EXCLUDED.space_total_capacity,
    space_seating_capacity = EXCLUDED.space_seating_capacity,
    space_type = EXCLUDED.space_type,
    space_building_level = EXCLUDED.space_building_level,
    space_link = EXCLUDED.space_link,
    space_accessibility_summary = EXCLUDED.space_accessibility_summary,
    space_accessibility_flags = EXCLUDED.space_accessibility_flags,
    space_description = EXCLUDED.space_description,
    modified_at = NOW()