UPDATE {{schema}}.event_date
SET
    release_status = $3,
    venue_uuid = $4::uuid,
    space_uuid = $5::uuid,
    start_date = $6,
    start_time = $7,
    end_date = $8,
    end_time = $9,
    entry_time = $10,
    duration = $11,
    all_day = COALESCE($12, all_day),
    modified_by = $13::uuid
WHERE uuid = $1::uuid
    AND event_uuid = $2::uuid