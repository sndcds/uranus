BEGIN;

-- Fail promptly rather than blocking live requests behind a long lock wait.
SET LOCAL lock_timeout = '5s';
LOCK TABLE uranus.venue IN ACCESS EXCLUSIVE MODE;

-- Check under the same lock as the DDL: concurrent writers cannot invalidate
-- the diagnosis. Never infer a replacement from org_uuid or permissions.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM uranus.venue
        WHERE scope NOT IN ('organization', 'shared') OR scope IS NULL
    ) THEN
        RAISE EXCEPTION 'Invalid venue.scope values exist; diagnose and classify manually before migration';
    END IF;
END
$$;

ALTER TABLE uranus.venue
    ALTER COLUMN scope DROP DEFAULT,
    ALTER COLUMN scope SET NOT NULL;

-- Replace only the known scope constraint from ddl/venue.ddl, after the
-- preflight. Other constraints are preserved, not guessed or dropped.
ALTER TABLE uranus.venue DROP CONSTRAINT IF EXISTS venue_scope_check;
ALTER TABLE uranus.venue
    ADD CONSTRAINT venue_scope_check CHECK (scope IN ('organization', 'shared'));

COMMIT;
