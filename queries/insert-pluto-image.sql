INSERT INTO {{schema}}.pluto_image
(file_name, gen_file_name, width, height, mime_type, exif,
alt_text, copyright, creator_name, license_id, description, focus_x, focus_y, user_id)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14) RETURNING id