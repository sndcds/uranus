SELECT jsonb_object_agg("group", perms) AS permissions_by_group
FROM (
    SELECT
        pb."group",
        jsonb_agg(
            jsonb_build_object(
                'label', pl.label,
                'description', pl.description,
                'bit', pb.bit
            ) ORDER BY pb.bit
        ) AS perms
FROM {{schema}}.permission_bit pb
JOIN {{schema}}.permission_label pl
ON pb."group" = pl."group"
AND pb.name = pl.name
WHERE pl.iso_639_1 = $1
GROUP BY pb."group"
) t