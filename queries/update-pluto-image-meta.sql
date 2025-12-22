UPDATE {{schema}}.pluto_image
SET
    alt_text = $1,
    copyright = $2,
    creator_name = $3,
    license_id = $4,
    description = $5,
    focus_x = $6,
    focus_y = $7
WHERE id = $8