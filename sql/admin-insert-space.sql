INSERT INTO {{schema}}.space (
    venue_uuid,
    name,
    description,
    space_type_id,
    building_level,
    total_capacity,
    seating_capacity,
    web_link,
    accessibility_flags,
    accessibility_summary
)
VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
RETURNING uuid