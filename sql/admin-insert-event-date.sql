INSERT INTO {{schema}}.event_date (
    uuid,
    event_uuid,
    venue_uuid,
    space_uuid,
    start_date,
    start_time,
    end_date,
    end_time,
    entry_time,
    duration,
    all_day,
    created_by
) VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6, $7, $8, $9, $10, $11, $12::uuid)