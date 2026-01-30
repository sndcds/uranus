UPDATE {{schema}}.event_date
SET
    venue_id = COALESCE($3, venue_id),
    start_date = $4,
    start_time = $5,
    end_date = $6,
    end_time = $7,
    entry_time = $8,
    duration = $9,
    all_day = COALESCE($10, all_day),
    modified_by = $11
WHERE id = $1 AND event_id = $2