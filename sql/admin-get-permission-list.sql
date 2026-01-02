SELECT jsonb_object_agg(group_id, perms) AS permissions_by_group
FROM (
    SELECT
        pb.group_id,
        jsonb_agg(
            jsonb_build_object(
                'label', pl.label,
                'description', pl.description,
                'bit', pb.bit
            ) ORDER BY pb.bit
        ) AS perms
FROM {{schema}}.permission_bit pb
JOIN {{schema}}.permission_label pl
ON pb.group_id = pl.group_id
AND pb.name = pl.name
WHERE pl.iso_639_1 = $1
GROUP BY pb.group_id
) t