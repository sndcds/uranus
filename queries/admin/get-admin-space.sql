SELECT
    name,
    description,
    space_type_id,
    building_level,
    total_capacity,
    seating_capacity,
    website_url,
    accessibility_flags,
    accessibility_summary
FROM {{schema}}.space
WHERE id = $1

