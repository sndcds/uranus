UPDATE {{schema}}.event_projection
SET search_vector =
    setweight(
        to_tsvector(
            'simple',
            concat_ws(
                ' ',
                title,
                subtitle,
                org_name,
                coalesce(array_to_string(tags, ' '), '')
            )
        ),
        'A'
    )

    ||

    setweight(
        to_tsvector(
            'simple',
            concat_ws(
                ' ',
                description,
                participation_info,
                meeting_point
            )
        ),
        'B'
    )

    ||

    setweight(
        to_tsvector(
            'simple',
            concat_ws(
                ' ',
                image_alt_text,
                image_description
            )
        ),
        'C'
    )

WHERE event_uuid = ANY($1::uuid[])