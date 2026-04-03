UPDATE {{schema}}.event_date
SET
    venue_uuid = $3::uuid,
    space_uuid = $4::uuid,
    start_date = $5,
    start_time = $6,
    end_date = $7,
    end_time = $8,
    entry_time = $9,
    duration = $10,
    all_day = COALESCE($11, all_day),
    modified_by = $12::uuid
WHERE uuid = $1::uuid AND event_uuid = $2::uuid