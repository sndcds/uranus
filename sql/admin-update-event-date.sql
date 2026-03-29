UPDATE {{schema}}.event_date
SET
    venue_uuid = $3,
    space_uuid = $4,
    start_date = $5,
    start_time = $6,
    end_date = $7,
    end_time = $8,
    entry_time = $9,
    duration = $10,
    all_day = COALESCE($11, all_day),
    modified_by = $12
WHERE uuid = $1 AND event_uuid = $2