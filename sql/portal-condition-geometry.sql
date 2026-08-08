AND (
    p.wkb_geometry IS NULL
    OR ST_Covers(
        p.wkb_geometry,
        COALESCE(edp.venue_point, ep.venue_point)
    )
)