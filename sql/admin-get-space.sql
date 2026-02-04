SELECT
    s.name,
    s.description,
    s.space_type_id,
    s.building_level,
    s.total_capacity,
    s.seating_capacity,
    s.website_link,
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
JOIN {{schema}}.venue v ON v.id = s.venue_id
JOIN {{schema}}.organization o ON o.id = v.organization_id
JOIN {{schema}}.user_organization_link uol ON uol.organization_id = o.id AND uol.user_id = $2
WHERE s.id = $1
