UPDATE {{schema}}.event_date_projection edp
SET search_vector =
    setweight(
        to_tsvector(
            'simple',
            concat_ws(
                ' ',
                coalesce(edp.venue_name, ep.venue_name),
                coalesce(edp.venue_street, ep.venue_street),
                coalesce(edp.venue_city, ep.venue_city)
                )
            ),
            'A'
        )
FROM {{schema}}.event_projection ep
WHERE ep.event_uuid = edp.event_uuid
    AND edp.event_date_uuid = ANY($1::uuid[])