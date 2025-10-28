UPDATE uranus.space
SET
    name,
    description,
    space_type_id,
    building_level,
    total_capacity,
    seating_capacity,
    website_url,
    accessibility_flags,
    accessibility_summary
WHERE id = $1

