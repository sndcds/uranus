UPDATE {{schema}}.space
SET
    name = $2,
    description = $3,
    space_type_id = $4,
    building_level = $5,
    total_capacity = $6,
    seating_capacity = $7,
    website_link = $8,
    accessibility_flags = $9,
    accessibility_summary = $10
WHERE id = $1;