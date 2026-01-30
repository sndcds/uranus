INSERT INTO {{schema}}.event_date (
    event_id,
    venue_id,
    start_date,
    start_time,
    end_date,
    end_time,
    entry_time,
    duration,
    all_day,
    created_by
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)