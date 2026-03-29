SELECT
    s.uuid,
    s.name,
    s.description,
    s.space_type,
    s.building_level,
    s.total_capacity,
    s.seating_capacity,
    s.web_link,
    s.accessibility_flags::text AS accessibility_flags,
    s.accessibility_summary,
    s.area_sqm,
    s.environmental_features,
    s.audio_features,
    s.presentation_features,
    s.lighting_features,
    s.climate_features,
    s.misc_features
FROM {{schema}}.space s
JOIN {{schema}}.venue v ON v.uuid = s.venue_uuid
JOIN {{schema}}.organization o ON o.uuid = v.org_uuid
JOIN {{schema}}.user_organization_link uol ON uol.org_uuid = o.uuid AND uol.user_uuid = $2
WHERE s.uuid = $1
