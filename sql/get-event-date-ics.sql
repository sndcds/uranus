SELECT
    edp.event_date_uuid,
    COALESCE(edp.venue_name, ep.venue_name),
    COALESCE(edp.venue_street, ep.venue_street),
    COALESCE(edp.venue_house_number, ep.venue_house_number),
    COALESCE(edp.venue_city, ep.venue_city),
    TO_CHAR(edp.start_date, 'YYYY-MM-DD') AS start_date,
    TO_CHAR(edp.start_time, 'HH24:MI') AS start_time,
    TO_CHAR(edp.end_date, 'YYYY-MM-DD') AS end_date,
    TO_CHAR(edp.end_time, 'HH24:MI') AS end_time,
    ep.title,
    ep.subtitle,
    ep.description AS description,
    ep.org_name,
    ep.org_contact_email
FROM {{schema}}.event_date_projection edp
JOIN {{schema}}.event_projection ep ON ep.event_uuid = edp.event_uuid
WHERE edp.event_date_uuid = $1::uuid