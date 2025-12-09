e.id AS id,
COALESCE(v_ed.id, v_ev.id) AS venue_id,
COALESCE(v_ed.name, v_ev.name) AS venue_name,
COALESCE(v_ed.city, v_ev.city) AS venue_city,
o.id AS organizer_id,
o.name AS organizer_name,
et_data.event_types