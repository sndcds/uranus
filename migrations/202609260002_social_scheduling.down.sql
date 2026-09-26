BEGIN;

-- Stop workers first. Do not leave scheduled work or unresolved attempts behind.
LOCK TABLE uranus.social_post_target, uranus.social_publication IN ACCESS EXCLUSIVE MODE;
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM uranus.social_post_target WHERE status IN ('scheduled', 'publishing')) THEN
        RAISE EXCEPTION 'cancel schedules and reconcile publications before rolling back Part 7';
    END IF;
END;
$$;
DROP INDEX uranus.social_post_target_due_idx;
ALTER TABLE uranus.social_post_target
    DROP COLUMN publication_source,
    DROP COLUMN publish_language;
ALTER TABLE uranus.social_publication DROP COLUMN publication_source;

COMMIT;
