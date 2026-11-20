-- Read-only deployment preflight. Does not classify or change any venue.
BEGIN READ ONLY;

SELECT scope, COUNT(*)
FROM uranus.venue
GROUP BY scope
ORDER BY scope;

SELECT COUNT(*) AS invalid_scope_count
FROM uranus.venue
WHERE scope NOT IN ('organization', 'shared')
   OR scope IS NULL;

SELECT column_default, is_nullable
FROM information_schema.columns
WHERE table_schema = 'uranus' AND table_name = 'venue' AND column_name = 'scope';

SELECT conname, convalidated, pg_get_constraintdef(oid) AS definition
FROM pg_constraint
WHERE conrelid = 'uranus.venue'::regclass AND contype = 'c'
ORDER BY conname;

COMMIT;
